package core_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stanleygomes/arkflix-games/app/internal/core"
	"github.com/stanleygomes/arkflix-games/app/internal/meta"
)

// writableExtra deixa a segunda pasta do ambiente falso ("Jogos antigos") gravável.
func writableExtra(t *testing.T, h harness) {
	t.Helper()
	cfg := h.svc.GetSettings()
	for i := range cfg.LibraryDirs {
		if cfg.LibraryDirs[i].Path == h.env.Extra {
			cfg.LibraryDirs[i].Writable = true
		}
	}
	if _, err := h.svc.SaveSettings(cfg); err != nil {
		t.Fatal(err)
	}
}

func exists(path string) bool { _, err := os.Lstat(path); return err == nil }

func leftovers(t *testing.T, dir string) {
	t.Helper()
	items, _ := os.ReadDir(dir)
	for _, it := range items {
		if strings.HasPrefix(it.Name(), ".arkflix-move-") {
			t.Fatalf("temporário sobrou: %s", it.Name())
		}
	}
}

func TestMoveGamesSameDisk(t *testing.T) {
	h := setup(t)
	smw := h.game(t, "Super Mario World")
	if err := h.svc.MoveGames("snes", h.env.Local, h.env.Extra); err == nil || !strings.Contains(err.Error(), "gravável") {
		t.Fatalf("destino só de leitura devia ser recusado: %v", err)
	}
	writableExtra(t, h)
	if _, err := h.svc.SetFavorite(smw.ID, true); err != nil {
		t.Fatal(err)
	}
	edited := "SMW editado"
	if _, err := h.svc.SaveMeta(smw.ID, meta.Override{Name: &edited, Rating: 5}); err != nil {
		t.Fatal(err)
	}
	l, _ := h.svc.CreateList("Plataforma")
	if _, err := h.svc.SetInList(l.ID, smw.ID, true); err != nil {
		t.Fatal(err)
	}

	plan, err := h.svc.MovePlan("snes", h.env.Local)
	if err != nil || len(plan) != 1 {
		t.Fatalf("plano: %+v %v", plan, err)
	}
	if p := plan[0]; p.Path != h.env.Extra || p.Count != 2 || p.Size <= 0 || !p.SameDisk || !p.Fits || p.Label != "Jogos antigos" {
		t.Fatalf("destino: %+v", p)
	}
	for _, bad := range [][2]string{{h.env.Local, h.env.Local}, {h.env.Local, t.TempDir()}, {"/", h.env.Extra}} {
		if err := h.svc.MoveGames("snes", bad[0], bad[1]); err == nil {
			t.Fatalf("%v devia ser recusado", bad)
		}
	}
	if err := h.svc.MoveGames("snes", h.env.Local, h.env.Extra); err != nil {
		t.Fatal(err)
	}
	ev := h.rec.waitTask(t, "move:snes")
	if ev.State != "done" || !strings.Contains(ev.Message, "2 jogo(s) movido(s)") {
		t.Fatalf("fim: %+v", ev)
	}
	for _, name := range []string{"Super Mario World (U) [!].zip", "Mega Man X (U) (V1.1).zip"} {
		if exists(filepath.Join(h.env.Local, "snes", name)) || !exists(filepath.Join(h.env.Extra, "snes", name)) {
			t.Fatalf("%s não foi movido", name)
		}
	}
	if !exists(filepath.Join(h.env.Extra, "snes", "screenshots", "Super Mario World (U) [!].png")) {
		t.Fatal("o print local vai junto com o jogo")
	}
	after := h.game(t, "SMW editado")
	if after.ID != smw.ID || after.Root != h.env.Extra || !after.Favorite || after.Rating != 5 || !after.HasSaves || after.Snap == "" {
		t.Fatalf("o jogo movido mantém id, favorito, metadados e saves: %+v", after)
	}
	lists, _ := h.svc.Lists()
	if len(lists) != 1 || len(lists[0].Games) != 1 || lists[0].Games[0] != smw.ID {
		t.Fatalf("lista: %+v", lists)
	}
	st := h.svc.Storage()
	for _, c := range st.ByConsole {
		if c.System == "snes" && (len(c.Roots) != 1 || c.Roots[0] != h.env.Extra) {
			t.Fatalf("pastas do console: %+v", c)
		}
	}
	if _, err := h.svc.MovePlan("snes", h.env.Extra); err != nil {
		t.Fatal(err)
	}
	if err := h.svc.MoveGames("snes", h.env.Local, h.env.Extra); err == nil {
		t.Fatal("sem jogos na origem devia avisar")
	}
}

func TestMoveGamesOtherDisk(t *testing.T) {
	h := setup(t)
	writableExtra(t, h)
	core.SetMoveOtherDisk(true)
	t.Cleanup(func() { core.SetMoveOtherDisk(false) })
	// mesmo nome no destino: pula e conta
	mmx := filepath.Join(h.env.Extra, "snes", "Mega Man X (U) (V1.1).zip")
	if err := os.MkdirAll(filepath.Dir(mmx), 0o755); err != nil {
		t.Fatal(err)
	}
	os.WriteFile(mmx, []byte("outro"), 0o644)
	src := filepath.Join(h.env.Local, "snes", "Super Mario World (U) [!].zip")
	want, _ := os.ReadFile(src)

	plan, _ := h.svc.MovePlan("snes", h.env.Local)
	if len(plan) != 1 || plan[0].SameDisk {
		t.Fatalf("outro disco: %+v", plan)
	}
	if err := h.svc.MoveGames("snes", h.env.Local, h.env.Extra); err != nil {
		t.Fatal(err)
	}
	ev := h.rec.waitTask(t, "move:snes")
	if ev.State != "done" || !strings.Contains(ev.Message, "1 jogo(s) movido(s)") || !strings.Contains(ev.Message, "1 pulado(s)") {
		t.Fatalf("fim: %+v", ev)
	}
	got, err := os.ReadFile(filepath.Join(h.env.Extra, "snes", "Super Mario World (U) [!].zip"))
	if err != nil || string(got) != string(want) || exists(src) {
		t.Fatalf("cópia: %v, origem ainda existe: %v", err, exists(src))
	}
	if !exists(filepath.Join(h.env.Local, "snes", "Mega Man X (U) (V1.1).zip")) {
		t.Fatal("o pulado fica onde estava")
	}
	leftovers(t, filepath.Join(h.env.Extra, "snes"))

	// pasta de jogo (vários arquivos) também
	if err := h.svc.MoveGames("psx", h.env.Local, h.env.Extra); err != nil {
		t.Fatal(err)
	}
	if ev := h.rec.waitTask(t, "move:psx"); ev.State != "done" {
		t.Fatalf("psx: %+v", ev)
	}
	if !exists(filepath.Join(h.env.Extra, "psx", "Crash Bandicoot (USA)", "Crash Bandicoot (USA).cue")) || exists(filepath.Join(h.env.Local, "psx", "Crash Bandicoot (USA)")) {
		t.Fatal("pasta do Crash não foi movida")
	}
}

func TestMoveGamesCancel(t *testing.T) {
	h := setup(t)
	writableExtra(t, h)
	core.SetMoveOtherDisk(true)
	core.SetMoveChunk(func(dst string) {
		if strings.Contains(dst, "Super Mario World") {
			h.svc.CancelTask("move:snes")
		}
	})
	t.Cleanup(func() { core.SetMoveOtherDisk(false); core.SetMoveChunk(nil) })
	if err := h.svc.MoveGames("snes", h.env.Local, h.env.Extra); err != nil {
		t.Fatal(err)
	}
	if ev := h.rec.waitTask(t, "move:snes"); ev.State != "cancelled" {
		t.Fatalf("devia cancelar: %+v", ev)
	}
	dir := filepath.Join(h.env.Extra, "snes")
	if !exists(filepath.Join(dir, "Mega Man X (U) (V1.1).zip")) || exists(filepath.Join(h.env.Local, "snes", "Mega Man X (U) (V1.1).zip")) {
		t.Fatal("o que já foi movido fica movido")
	}
	if exists(filepath.Join(dir, "Super Mario World (U) [!].zip")) || !exists(filepath.Join(h.env.Local, "snes", "Super Mario World (U) [!].zip")) {
		t.Fatal("o cancelado fica na origem")
	}
	leftovers(t, dir)
	if len(h.game(t, "Super Mario World").ID) == 0 || h.game(t, "Mega Man X").Root != h.env.Extra {
		t.Fatal("biblioteca varrida de novo depois de cancelar")
	}
}

func TestMoveGamesRefuses(t *testing.T) {
	h := setup(t)
	writableExtra(t, h)
	core.SetMoveOtherDisk(true)
	core.SetMoveFree(func(string) int64 { return 10 })
	t.Cleanup(func() { core.SetMoveOtherDisk(false); core.SetMoveFree(nil) })
	plan, _ := h.svc.MovePlan("snes", h.env.Local)
	if len(plan) != 1 || plan[0].Fits || plan[0].Free != 10 {
		t.Fatalf("sem espaço: %+v", plan)
	}
	if err := h.svc.MoveGames("snes", h.env.Local, h.env.Extra); err == nil || !strings.Contains(err.Error(), "falta espaço") {
		t.Fatalf("sem espaço devia recusar: %v", err)
	}
	core.SetMoveFree(nil)

	g := h.game(t, "Sonic The Hedgehog 2")
	if err := h.svc.Play(g.ID, ""); err != nil {
		t.Fatal(err)
	}
	if err := h.svc.MoveGames("snes", h.env.Local, h.env.Extra); err == nil || !strings.Contains(err.Error(), "feche o jogo") {
		t.Fatalf("com jogo aberto devia recusar: %v", err)
	}
	h.rec.waitGame(t, "exited")
	if !exists(filepath.Join(h.env.Local, "snes", "Super Mario World (U) [!].zip")) {
		t.Fatal("recusado não mexe em nada")
	}
}
