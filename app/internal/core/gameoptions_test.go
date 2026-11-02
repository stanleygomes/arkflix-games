package core_test

import (
	"testing"

	"github.com/stanleygomes/arkflix-games/app/internal/profiles"
)

func TestHidden(t *testing.T) {
	h := setup(t)
	g := h.game(t, "Mega Man X")
	v, err := h.svc.SetHidden(g.ID, true)
	if err != nil || !v.Hidden {
		t.Fatalf("ocultar: %+v %v", v, err)
	}
	if !h.game(t, "Mega Man X").Hidden {
		t.Fatal("oculto não persistiu")
	}
	// por perfil: outro perfil vê o jogo normal
	other, _ := h.svc.CreateProfile("Bia", "mushroom", profiles.Colors[2])
	if _, err := h.svc.SelectProfile(other.ID, ""); err != nil {
		t.Fatal(err)
	}
	if h.game(t, "Mega Man X").Hidden {
		t.Fatal("oculto vazou pra outro perfil")
	}
	if _, err := h.svc.SelectProfile(profiles.Main, ""); err != nil {
		t.Fatal(err)
	}
	if v, err := h.svc.SetHidden(g.ID, false); err != nil || v.Hidden {
		t.Fatalf("mostrar de novo: %+v %v", v, err)
	}
}
