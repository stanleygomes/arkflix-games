package saves_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stanleygomes/arkflix-games/app/internal/catalog"
	"github.com/stanleygomes/arkflix-games/app/internal/fixture"
	"github.com/stanleygomes/arkflix-games/app/internal/saves"
)

func game(t *testing.T, slug, launch string) saves.Game {
	t.Helper()
	sys, ok := catalog.Find(slug)
	if !ok {
		t.Fatal(slug)
	}
	return saves.Game{System: sys, Launch: launch}
}

func TestRetroArch(t *testing.T) {
	env := fixture.NewEnv(t.TempDir())
	l := saves.List(game(t, "snes", filepath.Join(env.Local, "snes", "Super Mario World (U) [!].zip")), env.Settings.Saves)
	if len(l.States) != 2 {
		t.Fatalf("esperava 2 states (o do hack não conta), veio %+v", l.States)
	}
	if l.States[0].Slot != -1 || !l.States[0].Thumb {
		t.Errorf("mais recente devia ser o automático com miniatura: %+v", l.States[0])
	}
	if l.States[1].Slot != 1 || l.States[1].Label != "Slot 1" {
		t.Errorf("slot 1: %+v", l.States[1])
	}
	if len(l.Files) != 1 || l.Files[0].Name != "Super Mario World (U) [!].srm" {
		t.Errorf("save de bateria: %+v", l.Files)
	}
	data, kind, err := saves.Thumbnail(l.States[1].Path)
	if err != nil || kind != "image/png" || len(data) == 0 {
		t.Errorf("miniatura: %v %s", err, kind)
	}
}

func TestPSPAndPS2(t *testing.T) {
	env := fixture.NewEnv(t.TempDir())
	psp := saves.List(game(t, "psp", filepath.Join(env.Local, "psp", "Lara Croft Tomb Raider - Legend (USA).iso")), env.Settings.Saves)
	if len(psp.States) != 1 || psp.States[0].Label != "Slot 1" || !psp.States[0].Thumb || len(psp.Files) != 1 {
		t.Fatalf("psp: %+v", psp)
	}
	ps2 := saves.List(game(t, "ps2", filepath.Join(env.Local, "ps2", "FIFA Soccer 07 (USA).iso")), env.Settings.Saves)
	if len(ps2.States) != 1 || ps2.States[0].Slot != 1 || !ps2.States[0].Thumb {
		t.Fatalf("ps2 states: %+v", ps2.States)
	}
	if len(ps2.Files) != 1 || !ps2.Files[0].Shared {
		t.Fatalf("memory card compartilhado: %+v", ps2.Files)
	}
	if _, kind, err := saves.Thumbnail(ps2.States[0].Path); err != nil || kind != "image/png" {
		t.Fatalf("miniatura de dentro do p2s: %v", err)
	}
	chd := saves.List(game(t, "ps2", "/x/jogo.chd"), env.Settings.Saves)
	if chd.Note == "" {
		t.Error("chd devia explicar por que não lista states")
	}
}

func TestDelete(t *testing.T) {
	env := fixture.NewEnv(t.TempDir())
	g := game(t, "snes", filepath.Join(env.Local, "snes", "Super Mario World (U) [!].zip"))
	st := saves.List(g, env.Settings.Saves).States[1]
	if err := saves.Delete(st.Path, env.Settings.Saves); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(st.Path + ".png"); !os.IsNotExist(err) {
		t.Error("miniatura devia sumir junto")
	}
	if err := saves.Delete("/etc/passwd", env.Settings.Saves); err == nil {
		t.Fatal("apagar fora das pastas de save devia ser recusado")
	}
	if err := saves.Delete(env.Settings.Saves.RetroArchStates+"/../../../retroarch.cfg", env.Settings.Saves); err == nil {
		t.Fatal("caminho com .. devia ser recusado")
	}
}
