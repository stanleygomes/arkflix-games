package core_test

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/stanleygomes/arkflix-games/app/internal/core"
	"github.com/stanleygomes/arkflix-games/app/internal/history"
	"github.com/stanleygomes/arkflix-games/app/internal/meta"
	"github.com/stanleygomes/arkflix-games/app/internal/profiles"
)

type clock struct {
	mu sync.Mutex
	t  time.Time
}

func (c *clock) now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.t
}

func (c *clock) set(t time.Time) {
	c.mu.Lock()
	c.t = t
	c.mu.Unlock()
}

// kidSetup: perfil infantil Nina com o Super Mario World liberado, relógio às
// 15h e o histórico de hoje com `played` de jogo (em duas sessões).
func kidSetup(t *testing.T, minutes int, bedtime string, played time.Duration) (harness, *clock, core.GameView, string) {
	t.Helper()
	h := setup(t)
	c := &clock{t: time.Date(2026, 3, 10, 15, 0, 0, 0, time.Local)}
	core.SetClock(h.svc, c.now)
	smw := h.game(t, "Super Mario World")
	if _, err := h.svc.SaveMeta(smw.ID, meta.Override{KidsOK: true}); err != nil {
		t.Fatal(err)
	}
	kid, err := h.svc.CreateProfile("Nina", "cat", profiles.Colors[2])
	if err != nil {
		t.Fatal(err)
	}
	if _, err := h.svc.SetProfileLimits(kid.ID, 60, ""); err == nil {
		t.Fatal("limite só vale pra perfil infantil")
	}
	if _, err := h.svc.SetProfileKids(kid.ID, true); err != nil {
		t.Fatal(err)
	}
	if v, err := h.svc.SetProfileLimits(kid.ID, minutes, bedtime); err != nil || v.DailyMinutes != minutes || v.Bedtime != bedtime {
		t.Fatalf("limites: %+v %v", v, err)
	}
	// ontem não conta; hoje, metade de manhã e metade à tarde
	var lines []byte
	today := c.now()
	for _, s := range []history.Session{
		{Game: smw.ID, Start: today.Add(-24 * time.Hour).Unix(), Seconds: 3 * 3600},
		{Game: smw.ID, Start: today.Add(-6 * time.Hour).Unix(), Seconds: int64(played.Seconds()) / 2},
		{Game: smw.ID, Start: today.Add(-2 * time.Hour).Unix(), Seconds: int64(played.Seconds()) - int64(played.Seconds())/2},
	} {
		b, _ := json.Marshal(s)
		lines = append(append(lines, b...), '\n')
	}
	if err := os.WriteFile(filepath.Join(h.env.Data, "profiles", kid.ID, "sessions.jsonl"), lines, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := h.svc.SelectProfile(kid.ID, ""); err != nil {
		t.Fatal(err)
	}
	return h, c, smw, kid.ID
}

func TestKidsDailyLimit(t *testing.T) {
	h, _, _, kid := kidSetup(t, 60, "", 35*time.Minute)
	info := h.svc.KidsTime()
	if !info.Limited || info.MinutesLeft != 25 || info.PlayedToday != 35 || info.Reason != "" {
		t.Fatalf("tempo de hoje: %+v", info)
	}
	if _, err := h.svc.SetProfileLimits(kid, 0, ""); !errors.Is(err, core.ErrKids) {
		t.Fatalf("a criança não muda o próprio limite: %v", err)
	}
	// zera: Play recusa com o motivo
	h2, _, smw, _ := kidSetup(t, 60, "", 61*time.Minute)
	if info := h2.svc.KidsTime(); info.MinutesLeft != 0 || info.Reason != core.ErrTimeUp.Error() {
		t.Fatalf("sem tempo: %+v", info)
	}
	if err := h2.svc.Play(smw.ID, ""); !errors.Is(err, core.ErrTimeUp) {
		t.Fatalf("Play devia recusar: %v", err)
	}
}

func TestKidsBedtime(t *testing.T) {
	h, c, smw, _ := kidSetup(t, 0, "21:00", 0)
	c.set(time.Date(2026, 3, 10, 20, 0, 0, 0, time.Local))
	if info := h.svc.KidsTime(); !info.Limited || info.MinutesLeft != 60 || info.Bedtime != "21:00" {
		t.Fatalf("antes de dormir: %+v", info)
	}
	for _, at := range []time.Time{
		time.Date(2026, 3, 10, 21, 30, 0, 0, time.Local),
		time.Date(2026, 3, 11, 2, 0, 0, 0, time.Local), // madrugada ainda é noite
	} {
		c.set(at)
		if err := h.svc.Play(smw.ID, ""); !errors.Is(err, core.ErrBedtime) {
			t.Fatalf("%s: Play devia recusar: %v", at, err)
		}
	}
}

func TestKidsWatcherClosesGame(t *testing.T) {
	core.SetKidsTimers(20*time.Millisecond, 5*time.Minute)
	t.Cleanup(func() { core.SetKidsTimers(30*time.Second, 5*time.Minute) })
	core.SetQuitGrace(100 * time.Millisecond)
	h, c, smw, _ := kidSetup(t, 60, "", 57*time.Minute)
	t.Setenv("ARKFLIX_FAKE_SLEEP", "30")
	if err := h.svc.Play(smw.ID, ""); err != nil {
		t.Fatal(err)
	}
	h.rec.waitGame(t, "running")
	warn := waitLimit(t, h.rec, func(e core.LimitEvent) bool { return e.MinutesLeft > 0 })
	if warn.MinutesLeft != 3 || warn.Game != smw.ID {
		t.Fatalf("aviso: %+v", warn)
	}
	// o jogo aberto conta: 2 min depois, restam 1
	c.set(c.now().Add(2 * time.Minute))
	if info := h.svc.KidsTime(); info.MinutesLeft != 1 || info.PlayedToday != 59 {
		t.Fatalf("com o jogo aberto: %+v", info)
	}
	c.set(c.now().Add(2 * time.Minute))
	waitLimit(t, h.rec, func(e core.LimitEvent) bool { return e.MinutesLeft == 0 })
	if ev := h.rec.waitGame(t, "exited"); ev.Error != "" {
		t.Fatalf("fechou com erro: %+v", ev)
	}
}

func waitLimit(t *testing.T, r *recorder, ok func(core.LimitEvent) bool) core.LimitEvent {
	t.Helper()
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		r.mu.Lock()
		for i, e := range r.events {
			if ev, is := r.data[i].(core.LimitEvent); is && e == "limit" && ok(ev) {
				r.mu.Unlock()
				return ev
			}
		}
		r.mu.Unlock()
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatal("evento limit não chegou")
	return core.LimitEvent{}
}
