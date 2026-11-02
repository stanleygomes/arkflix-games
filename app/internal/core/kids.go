package core

import (
	"errors"
	"fmt"
	"time"

	"github.com/stanleygomes/arkflix-games/app/internal/catalog"
	"github.com/stanleygomes/arkflix-games/app/internal/history"
	"github.com/stanleygomes/arkflix-games/app/internal/profiles"
)

// Limite de tempo do perfil infantil: minutos por dia e hora de dormir.
// Um vigia olha o jogo aberto, avisa faltando pouco e fecha (salvando) no fim.

var (
	// kidsTick: de quanto em quanto tempo o vigia confere o relógio.
	kidsTick = 30 * time.Second
	// kidsWarn: com quanto tempo restando vai o aviso.
	kidsWarn = 5 * time.Minute
)

// wakeHour: até esta hora da manhã ainda vale a hora de dormir da noite anterior.
const wakeHour = 6

var (
	ErrTimeUp  = errors.New("Acabou o tempo de hoje. Volta amanhã!")
	ErrBedtime = errors.New("Já passou da hora de dormir")
)

// limits é o que o adulto definiu pro perfil infantil.
type limits struct {
	daily   int    // minutos por dia (0 = sem limite)
	bedtime string // "HH:MM" (vazio = sem)
}

func (l limits) on() bool { return l.daily > 0 || l.bedtime != "" }

// KidsTimeInfo é o tempo de jogo do perfil em uso.
type KidsTimeInfo struct {
	Limited     bool   `json:"limited"`     // tem limite (só perfil infantil)
	MinutesLeft int    `json:"minutesLeft"` // quanto ainda dá pra jogar hoje
	Bedtime     string `json:"bedtime"`
	PlayedToday int    `json:"playedToday"` // minutos
	Reason      string `json:"reason"`      // por que não dá pra jogar ("" = dá)
}

// LimitEvent: aviso do vigia (evento "limit"). MinutesLeft 0 = fechou o jogo.
type LimitEvent struct {
	Game        string `json:"game"`
	MinutesLeft int    `json:"minutesLeft"`
	Message     string `json:"message"`
}

func (s *Service) now() time.Time {
	if s.opt.Now != nil {
		return s.opt.Now()
	}
	return time.Now()
}

// kidsLeft: quanto ainda dá pra jogar e, se nada, o motivo. ok=false = sem limite.
func (s *Service) kidsLeft(u *user) (left time.Duration, reason error, ok bool) {
	if !u.kids || !u.limits.on() {
		return 0, nil, false
	}
	now := s.now()
	left = time.Duration(1<<62 - 1)
	if u.limits.daily > 0 {
		left = time.Duration(u.limits.daily)*time.Minute - s.playedToday(u, now)
		if left <= 0 {
			return 0, ErrTimeUp, true
		}
	}
	if u.limits.bedtime != "" {
		bt, _ := time.Parse("15:04", u.limits.bedtime)
		y, m, d := now.Date()
		bed := time.Date(y, m, d, bt.Hour(), bt.Minute(), 0, 0, now.Location())
		if now.Hour() < wakeHour || !now.Before(bed) {
			return 0, ErrBedtime, true
		}
		left = min(left, bed.Sub(now))
	}
	return left, nil, true
}

// playedToday: sessões gravadas do dia (local) mais o jogo aberto.
func (s *Service) playedToday(u *user, now time.Time) time.Duration {
	y, m, d := now.Date()
	midnight := time.Date(y, m, d, 0, 0, 0, 0, now.Location())
	sessions, _ := u.hist.All()
	played := history.PlayedBetween(sessions, midnight, now)
	s.mu.Lock()
	g := s.game
	s.mu.Unlock()
	if g != nil && g.user == u {
		played += now.Sub(later(g.started, midnight))
	}
	return played
}

func later(a, b time.Time) time.Time {
	if a.After(b) {
		return a
	}
	return b
}

func ceilMinutes(d time.Duration) int { return int((d + time.Minute - 1) / time.Minute) }

// KidsTime: limite e tempo restante do perfil em uso.
func (s *Service) KidsTime() KidsTimeInfo {
	u := s.cur()
	left, reason, ok := s.kidsLeft(u)
	if !ok {
		return KidsTimeInfo{}
	}
	info := KidsTimeInfo{Limited: true, Bedtime: u.limits.bedtime, MinutesLeft: ceilMinutes(left),
		PlayedToday: int(s.playedToday(u, s.now()) / time.Minute)}
	if reason != nil {
		info.Reason = reason.Error()
	}
	return info
}

// SetProfileLimits define o tempo por dia (minutos, 0 = sem limite) e a hora
// de dormir ("HH:MM", vazio = sem) de um perfil infantil. Só um adulto muda.
func (s *Service) SetProfileLimits(id string, minutes int, bedtime string) (profiles.View, error) {
	if err := s.adult(); err != nil {
		return profiles.View{}, err
	}
	p, err := s.profiles.Get(id)
	if err != nil {
		return profiles.View{}, err
	}
	if !p.Kids {
		return profiles.View{}, errors.New("limite de tempo só no perfil infantil")
	}
	p, err = s.profiles.SetLimits(id, minutes, bedtime)
	return p.View(), err
}

// watchKids fica de olho no jogo aberto do perfil infantil até ele fechar.
func (s *Service) watchKids(u *user, g *gameRun, done <-chan struct{}) {
	t := time.NewTicker(kidsTick)
	defer t.Stop()
	warned := false
	for {
		select {
		case <-done:
			return
		case <-t.C:
		}
		left, reason, ok := s.kidsLeft(u)
		if !ok {
			return
		}
		if reason != nil || left <= 0 {
			msg := ErrTimeUp.Error()
			if reason != nil {
				msg = reason.Error()
			}
			s.gameMessage(g, msg+" Salvando e fechando o jogo.")
			s.opt.Emitter.Emit("limit", LimitEvent{Game: g.id, Message: msg})
			s.quitGame(g)
			return
		}
		if !warned && left <= kidsWarn {
			warned = true
			n := ceilMinutes(left)
			msg := fmt.Sprintf("Faltam %d min de jogo hoje. Hora de salvar!", n)
			s.gameMessage(g, msg)
			s.opt.Emitter.Emit("limit", LimitEvent{Game: g.id, MinutesLeft: n, Message: msg})
		}
	}
}

// gameMessage mostra um aviso dentro do jogo (só o RetroArch sabe).
func (s *Service) gameMessage(g *gameRun, msg string) {
	if g.emulator == catalog.RetroArch && guideCommands() {
		raCommand("SHOW_MSG " + msg)
	}
}
