package core_test

import (
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

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

func TestPlayPS1WithoutBiosUsesHLECore(t *testing.T) {
	h := setup(t)
	os.Remove(fixture.PS1BiosPath(h.env.Settings))
	// com o PCSX ReARMed instalado, o PS1 abre mesmo sem a BIOS da Sony
	os.WriteFile(filepath.Join(h.env.Cores, "pcsx_rearmed_libretro.so"), []byte("core"), 0o644)
	g := h.game(t, "Crash Bandicoot")
	if err := h.svc.Play(g.ID, ""); err != nil {
		t.Fatal(err)
	}
	h.rec.waitGame(t, "exited")
	if !strings.Contains(h.env.Calls(), "pcsx_rearmed_libretro.so") {
		t.Fatalf("devia usar o core que imita a BIOS: %s", h.env.Calls())
	}
}

func TestBiosFromServer(t *testing.T) {
	h := setup(t)
	os.Remove(fixture.PS1BiosPath(h.env.Settings))
	os.Remove(fixture.PS2BiosPath(h.env.Settings))
	fake := fixture.NewFakeRomM()
	fake.PutFirmware("psx", "scph5501.bin", make([]byte, 512<<10))
	fake.PutFirmware("ps2", "SCPH-70012.bin", fixture.PS2Bios())
	fake.PutFirmware("ps2", "leiame.txt", []byte("não é BIOS"))
	srv := httptest.NewServer(fake)
	t.Cleanup(srv.Close)
	if _, err := h.svc.ConnectRemote(srv.URL, "stanley", "senha123"); err != nil {
		t.Fatal(err)
	}
	// a lista do servidor chega e a BIOS vem junto, sem a pessoa fazer nada
	deadline := time.Now().Add(10 * time.Second)
	for !biosOf(t, h.svc, "psx").OK || !biosOf(t, h.svc, "ps2").OK {
		if time.Now().After(deadline) {
			t.Fatalf("BIOS não veio do servidor: %+v %+v", biosOf(t, h.svc, "psx"), biosOf(t, h.svc, "ps2"))
		}
		time.Sleep(50 * time.Millisecond)
	}
	if _, err := os.Stat(filepath.Join(biosOf(t, h.svc, "ps2").Dir, "leiame.txt")); err == nil {
		t.Fatal("baixou arquivo que não é BIOS")
	}
	if ev := h.rec.waitTask(t, "bios"); ev.State != "done" || !strings.Contains(ev.Message, "2 arquivo") {
		t.Fatalf("aviso: %+v", ev)
	}
}
