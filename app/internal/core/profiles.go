package core

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/stanleygomes/arkflix-games/app/internal/history"
	"github.com/stanleygomes/arkflix-games/app/internal/input"
	"github.com/stanleygomes/arkflix-games/app/internal/lists"
	"github.com/stanleygomes/arkflix-games/app/internal/profiles"
	"github.com/stanleygomes/arkflix-games/app/internal/settings"
)

// user é o que muda de um perfil pro outro: favoritos e tempo de jogo
// (state.json), acessibilidade, histórico, listas, controle e saves do RetroArch.
type user struct {
	id     string
	dir    string
	store  *settings.Store
	hist   *history.Log
	lists  *lists.Store
	inputs input.Store
	kids   bool           // perfil infantil: só jogos liberados, nada de configurar
	plays  map[string]int // mexido só com Service.mu
}

// ErrKids: o perfil infantil não mexe em configurações, importação, saves e
// metadados (a interface esconde; aqui é a trava de verdade).
var ErrKids = errors.New("bloqueado no perfil infantil")

func (u *user) allows(e entry) bool { return !u.kids || e.edit.KidsOK }

func (s *Service) adult() error {
	if s.cur().kids {
		return ErrKids
	}
	return nil
}

func (u *user) raSaves() string  { return filepath.Join(u.dir, "saves", "retroarch", "saves") }
func (u *user) raStates() string { return filepath.Join(u.dir, "saves", "retroarch", "states") }
func (u *user) raSavesCfg() string {
	return filepath.Join(u.dir, "retroarch-saves.cfg")
}

// raAppend: configs extras do RetroArch pro perfil (controle e, fora do
// principal, as pastas de save próprias).
func (u *user) raAppend() []string {
	out := []string{u.inputs.ConfigPath()}
	if u.id != profiles.Main {
		out = append(out, u.raSavesCfg())
	}
	return out
}

func (s *Service) cur() *user {
	s.umu.RLock()
	defer s.umu.RUnlock()
	return s.u
}

// loadUser abre os dados do perfil, criando o que faltar.
func (s *Service) loadUser(id string) (*user, error) {
	p, err := s.profiles.Get(id)
	if err != nil {
		return nil, err
	}
	dir := s.profiles.DataDir(id)
	store := s.opt.Store
	if id != profiles.Main {
		// perfil novo começa com a acessibilidade do principal
		if store, err = settings.Open(dir, s.opt.Store.Settings()); err != nil {
			return nil, err
		}
	}
	u := &user{
		id: id, dir: dir, store: store, kids: p.Kids,
		hist:   &history.Log{Path: filepath.Join(dir, "sessions.jsonl"), MinPlay: s.opt.MinPlay},
		lists:  &lists.Store{Path: filepath.Join(dir, "lists.json")},
		inputs: input.Store{Dir: dir},
	}
	sessions, err := u.hist.All()
	if err != nil {
		return nil, err
	}
	u.plays = u.hist.Plays(sessions)
	m, err := u.inputs.Load()
	if err != nil {
		m = input.Default()
	}
	if err := u.inputs.WriteConfig(m); err != nil {
		return nil, err
	}
	if id != profiles.Main {
		for _, d := range []string{u.raSaves(), u.raStates()} {
			if err := os.MkdirAll(d, 0o700); err != nil {
				return nil, err
			}
		}
		cfg := fmt.Sprintf("savefile_directory = %q\nsavestate_directory = %q\n", u.raSaves(), u.raStates())
		if err := os.WriteFile(u.raSavesCfg(), []byte(cfg), 0o600); err != nil {
			return nil, err
		}
	}
	return u, nil
}

// saveDirs: pastas de save do perfil em uso. PSP e PS2 são compartilhados
// (os emuladores não têm como trocar a pasta por linha de comando).
func (s *Service) saveDirs() settings.SaveDirs {
	d := s.opt.Store.Settings().Saves
	if u := s.cur(); u != nil && u.id != profiles.Main {
		d.RetroArchSaves, d.RetroArchStates = u.raSaves(), u.raStates()
	}
	return d
}

// ProfileState é a lista de perfis, o atual e se a tela "Quem vai jogar?" aparece.
type ProfileState struct {
	Profiles []profiles.View `json:"profiles"`
	Current  string          `json:"current"`
	Choose   bool            `json:"choose"`
	Setup    bool            `json:"setup"` // primeira vez: configurar o perfil principal
	Avatars  []string        `json:"avatars"`
	Colors   []string        `json:"colors"`
}

// Profiles lista os perfis.
func (s *Service) Profiles() (ProfileState, error) {
	all, err := s.profiles.All()
	if err != nil {
		return ProfileState{}, err
	}
	out := ProfileState{Current: s.cur().id, Avatars: profiles.Avatars, Colors: profiles.Colors}
	for _, p := range all {
		out.Profiles = append(out.Profiles, p.View())
		if p.ID == profiles.Main && p.New {
			out.Setup = true
		}
	}
	s.umu.RLock()
	out.Choose = len(all) > 1 && !s.chosen
	s.umu.RUnlock()
	return out, nil
}

// SelectProfile troca o perfil em uso (pede o PIN se ele tiver) e devolve o
// estado inicial do perfil.
func (s *Service) SelectProfile(id, pin string) (Boot, error) {
	if s.Running() != "" {
		return Boot{}, errors.New("feche o jogo antes de trocar de perfil")
	}
	if err := s.profiles.Check(id, pin); err != nil {
		if errors.Is(err, profiles.ErrPIN) {
			time.Sleep(400 * time.Millisecond) // chute de PIN fica lento
		}
		return Boot{}, err
	}
	u, err := s.loadUser(id)
	if err != nil {
		return Boot{}, err
	}
	s.mu.Lock()
	s.umu.Lock()
	s.u, s.chosen = u, true
	s.umu.Unlock()
	s.mu.Unlock()
	s.scan() // "tem save" depende do perfil
	return s.Bootstrap()
}

// CreateProfile cria um perfil.
func (s *Service) CreateProfile(name, avatar, color string) (profiles.View, error) {
	if err := s.adult(); err != nil {
		return profiles.View{}, err
	}
	p, err := s.profiles.Create(name, avatar, color)
	return p.View(), err
}

// UpdateProfile troca nome, avatar e cor.
func (s *Service) UpdateProfile(id, name, avatar, color string) (profiles.View, error) {
	if err := s.adult(); err != nil {
		return profiles.View{}, err
	}
	p, err := s.profiles.Update(id, name, avatar, color)
	return p.View(), err
}

// SetProfilePIN define, troca ou (next vazio) remove o PIN; pede o atual.
func (s *Service) SetProfilePIN(id, current, next string) (profiles.View, error) {
	if err := s.adult(); err != nil {
		return profiles.View{}, err
	}
	p, err := s.profiles.SetPIN(id, current, next)
	return p.View(), err
}

// DeleteProfile apaga um perfil (não o principal nem o que está em uso).
// A pasta dele vai pra profiles/apagados/.
func (s *Service) DeleteProfile(id, pin string) error {
	if err := s.adult(); err != nil {
		return err
	}
	if id == s.cur().id {
		return errors.New("troque de perfil antes de apagar este")
	}
	return s.profiles.Delete(id, pin)
}

// CheckProfilePIN confere o PIN (perfil sem PIN aceita qualquer coisa).
func (s *Service) CheckProfilePIN(id, pin string) error {
	err := s.profiles.Check(id, pin)
	if errors.Is(err, profiles.ErrPIN) {
		time.Sleep(400 * time.Millisecond)
	}
	return err
}

// SetProfileKids marca ou desmarca um perfil como infantil (não o em uso).
func (s *Service) SetProfileKids(id string, kids bool) (profiles.View, error) {
	if err := s.adult(); err != nil {
		return profiles.View{}, err
	}
	if id == s.cur().id {
		return profiles.View{}, errors.New("troque de perfil antes de mudar este")
	}
	p, err := s.profiles.SetKids(id, kids)
	return p.View(), err
}
