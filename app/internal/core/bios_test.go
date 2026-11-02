package core_test

import (
	"os"
	"strings"
	"testing"

	"github.com/stanleygomes/arkflix-games/app/internal/core"
	"github.com/stanleygomes/arkflix-games/app/internal/fixture"
)

func biosOf(t *testing.T, svc *core.Service, system string) core.BiosInfo {
	t.Helper()
	for _, b := range svc.BiosStatus() {
		if b.System == system {
			return b
		}
	}
	t.Fatalf("sem BIOS de %s", system)
	return core.BiosInfo{}
}

func TestBiosStatus(t *testing.T) {
	h := setup(t)
	ps1, ps2 := biosOf(t, h.svc, "psx"), biosOf(t, h.svc, "ps2")
	if !ps1.Required || !ps1.OK || ps1.Found[0] != "scph5501.bin" || !strings.HasSuffix(ps1.Dir, "retroarch/system") {
		t.Fatalf("PS1: %+v", ps1)
	}
	if !ps2.OK || !strings.HasSuffix(ps2.Dir, "PCSX2/bios") {
		t.Fatalf("PS2: %+v", ps2)
	}
	if ds := biosOf(t, h.svc, "nds"); ds.Required || ds.OK {
		t.Fatalf("DS: %+v", ds)
	}
	os.Remove(fixture.PS2BiosPath(h.env.Settings))
	if ps2 := biosOf(t, h.svc, "ps2"); ps2.OK || len(ps2.Missing) != 1 {
		t.Fatalf("PS2 sem BIOS: %+v", ps2)
	}
}

func TestPlayWithoutBios(t *testing.T) {
	h := setup(t)
	os.Remove(fixture.PS1BiosPath(h.env.Settings))
	g := h.game(t, "Crash Bandicoot")
	err := h.svc.Play(g.ID, "")
	if err == nil || !strings.Contains(err.Error(), "Falta a BIOS do PlayStation (scph5501.bin etc.): coloque em ") ||
		!strings.Contains(err.Error(), "retroarch/system") {
		t.Fatalf("erro: %v", err)
	}
	if h.env.Calls() != "" {
		t.Fatalf("não devia abrir o emulador: %s", h.env.Calls())
	}
}

func TestOpenBiosFolder(t *testing.T) {
	h := setup(t)
	dir := biosOf(t, h.svc, "nds").Dir
	os.RemoveAll(dir)
	// sem janela (modo web) não abre, mas a pasta já fica criada
	if err := h.svc.OpenBiosFolder("nds"); err == nil {
		t.Fatal("abrir pasta sem desktop devia falhar")
	}
	if st, err := os.Stat(dir); err != nil || !st.IsDir() {
		t.Fatalf("pasta não criada: %v", err)
	}
	if err := h.svc.OpenBiosFolder("snes"); err == nil {
		t.Fatal("SNES não usa BIOS")
	}
}
