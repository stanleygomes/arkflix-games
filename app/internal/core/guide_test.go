package core_test

import (
	"testing"
	"time"

	"github.com/stanleygomes/arkflix-games/app/internal/core"
)

func TestGuideQuitsGame(t *testing.T) {
	h := setup(t)
	t.Setenv("ARKFLIX_FAKE_SLEEP", "30")
	core.SetQuitGrace(100 * time.Millisecond)
	if err := h.svc.GameCommand("quit"); err == nil {
		t.Fatal("sem jogo aberto devia dar erro")
	}
	g := h.game(t, "Super Mario World")
	if err := h.svc.Play(g.ID, ""); err != nil {
		t.Fatal(err)
	}
	h.rec.waitGame(t, "running")
	if got := h.svc.Guide(); got.Game != g.ID {
		t.Fatalf("guia: %+v", got)
	}
	if err := h.svc.GameCommand("pular"); err == nil {
		t.Fatal("comando desconhecido devia dar erro")
	}
	start := time.Now()
	if err := h.svc.GameCommand("quit"); err != nil {
		t.Fatal(err)
	}
	ev := h.rec.waitGame(t, "exited")
	if ev.Error != "" || time.Since(start) > 5*time.Second {
		t.Fatalf("fechar pelo guia: %+v em %s", ev, time.Since(start))
	}
	if got := h.svc.Guide(); got.Game != "" {
		t.Fatalf("jogo fechado ainda no guia: %+v", got)
	}
}

func TestCancelTask(t *testing.T) {
	h := setup(t)
	if err := h.svc.CancelTask("nada"); err == nil {
		t.Fatal("tarefa inexistente devia dar erro")
	}
}
