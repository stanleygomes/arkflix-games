package meta_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stanleygomes/arkflix-games/app/internal/fixture"
	"github.com/stanleygomes/arkflix-games/app/internal/library"
	"github.com/stanleygomes/arkflix-games/app/internal/meta"
)

func games(t *testing.T, env fixture.Env) map[string]library.Game {
	t.Helper()
	out := map[string]library.Game{}
	for _, g := range library.Scan(env.Settings.LibraryDirs) {
		out[g.Title] = g
	}
	return out
}

func TestGamelistAndESDECover(t *testing.T) {
	env := fixture.NewEnv(t.TempDir())
	l := &meta.Local{ESDEDir: func() string { return env.ESDE }}
	info := l.Lookup(games(t, env)["Super Mario World"])
	if info.Name != "Super Mario World" || info.Year != 1990 || info.Players != "1-2" || !strings.Contains(info.Summary, "Yoshi") {
		t.Fatalf("gamelist: %+v", info)
	}
	if !strings.Contains(info.Cover, "downloaded_media") {
		t.Fatalf("capa do ES-DE: %q", info.Cover)
	}
	if !strings.HasSuffix(info.Manual, ".pdf") || !strings.Contains(info.Snap, "screenshots") {
		t.Fatalf("manual e print locais: %+v", info)
	}
	mmx := l.Lookup(games(t, env)["Mega Man X"])
	if len(mmx.Genres) != 2 || mmx.Cover != "" {
		t.Fatalf("mmx: %+v", mmx)
	}
}

func TestCoverNextToRoms(t *testing.T) {
	env := fixture.NewEnv(t.TempDir())
	l := &meta.Local{ESDEDir: func() string { return "" }}
	info := l.Lookup(games(t, env)["Sonic The Hedgehog 2"])
	if filepath.Base(filepath.Dir(info.Cover)) != "covers" || info.Name != "" {
		t.Fatalf("capa da pasta covers/: %+v", info)
	}
	if filepath.Base(filepath.Dir(info.Video)) != "videos" {
		t.Fatalf("vídeo da pasta videos/: %+v", info)
	}
}

func TestRefresh(t *testing.T) {
	env := fixture.NewEnv(t.TempDir())
	l := &meta.Local{ESDEDir: func() string { return env.ESDE }}
	g := games(t, env)["Crash Bandicoot"]
	if l.Lookup(g).Name != "" {
		t.Fatal("crash não tem gamelist ainda")
	}
	os.MkdirAll(filepath.Join(env.ESDE, "gamelists", "psx"), 0o755)
	os.WriteFile(filepath.Join(env.ESDE, "gamelists", "psx", "gamelist.xml"),
		[]byte(`<gameList><game><path>./Crash Bandicoot (USA)</path><name>Crash Bandicoot</name><releasedate>19960909T000000</releasedate></game></gameList>`), 0o644)
	if l.Lookup(g).Name != "" {
		t.Fatal("sem Refresh devia usar o cache")
	}
	l.Refresh()
	if got := l.Lookup(g); got.Name != "Crash Bandicoot" || got.Year != 1996 {
		t.Fatalf("depois do refresh: %+v", got)
	}
}

func TestBrokenGamelist(t *testing.T) {
	env := fixture.NewEnv(t.TempDir())
	os.WriteFile(filepath.Join(env.ESDE, "gamelists", "snes", "gamelist.xml"), []byte("<gameList><game>"), 0o644)
	l := &meta.Local{ESDEDir: func() string { return env.ESDE }}
	if info := l.Lookup(games(t, env)["Super Mario World"]); info.Name != "" {
		t.Fatalf("xml quebrado não devia gerar dado: %+v", info)
	}
}
