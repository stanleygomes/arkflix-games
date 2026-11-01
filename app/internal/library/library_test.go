package library_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stanleygomes/arkflix-games/app/internal/library"
	"github.com/stanleygomes/arkflix-games/app/internal/settings"
)

func touch(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestCleanTitle(t *testing.T) {
	cases := map[string]string{
		"Super Mario World (U) [!].zip":                     "Super Mario World",
		"Sonic The Hedgehog 2 (World) (Rev A).zip":          "Sonic The Hedgehog 2",
		"1571 - Tomb Raider III (USA)":                      "Tomb Raider III",
		"New Super Mario Bros..zip":                         "New Super Mario Bros.",
		"mspacman.zip":                                      "mspacman",
		"Lara Croft Tomb Raider - Legend (USA) (En,Fr).iso": "Lara Croft Tomb Raider - Legend",
		"(Proto).nes":                                       "(Proto)",
	}
	for in, want := range cases {
		if got := library.CleanTitle(in); got != want {
			t.Errorf("CleanTitle(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestScan(t *testing.T) {
	local, mount := t.TempDir(), t.TempDir()
	touch(t, filepath.Join(local, "snes", "Super Mario World (U) [!].zip"))
	touch(t, filepath.Join(local, "snes", ".rsync-tmp.zip"))
	touch(t, filepath.Join(local, "snes", "notes.txt"))
	touch(t, filepath.Join(local, "megadrive", "Sonic.bin"))
	touch(t, filepath.Join(local, "psx", "Crash Bandicoot (USA)", "Crash Bandicoot (USA).cue"))
	touch(t, filepath.Join(local, "psx", "Crash Bandicoot (USA)", "Crash Bandicoot (USA).bin"))
	touch(t, filepath.Join(local, "psx", "Vazio", "leia.txt"))
	touch(t, filepath.Join(local, "desconhecido", "jogo.rom"))
	// mesmo jogo no mount: vale o local
	touch(t, filepath.Join(mount, "snes", "Super Mario World (U) [!].zip"))
	touch(t, filepath.Join(mount, "nes", "Super Mario Bros. 3.zip"))

	games := library.Scan([]settings.LibraryDir{
		{Path: local, Writable: true},
		{Path: mount},
		{Path: filepath.Join(t.TempDir(), "nao-existe")},
	})
	byTitle := map[string]library.Game{}
	for _, g := range games {
		byTitle[g.Title] = g
	}
	if len(games) != 4 {
		t.Fatalf("esperava 4 jogos, veio %d: %+v", len(games), games)
	}
	smw := byTitle["Super Mario World"]
	if !smw.Local || smw.Root != local {
		t.Errorf("SMW devia vir da raiz local: %+v", smw)
	}
	crash := byTitle["Crash Bandicoot"]
	if filepath.Ext(crash.Launch) != ".cue" {
		t.Errorf("Crash devia abrir pelo .cue: %s", crash.Launch)
	}
	if byTitle["Sonic"].System != "genesis" {
		t.Errorf("pasta megadrive devia virar genesis: %+v", byTitle["Sonic"])
	}
	if games[0].System != "nes" {
		t.Errorf("ordem dos consoles: primeiro devia ser nes, veio %s", games[0].System)
	}
}

func TestIDStable(t *testing.T) {
	if library.ID("snes", "Game.zip") != library.ID("snes", "game.ZIP") {
		t.Error("ID devia ignorar maiúsculas")
	}
	if library.ID("snes", "Game.zip") == library.ID("nes", "Game.zip") {
		t.Error("consoles diferentes não podem ter o mesmo ID")
	}
}
