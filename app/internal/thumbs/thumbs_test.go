package thumbs_test

import (
	"context"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stanleygomes/arkflix-games/app/internal/fixture"
	"github.com/stanleygomes/arkflix-games/app/internal/library"
	"github.com/stanleygomes/arkflix-games/app/internal/meta"
	"github.com/stanleygomes/arkflix-games/app/internal/thumbs"
)

func TestParseDAT(t *testing.T) {
	dat := `clrmamepro (
	name "Sistema"
)

game (
	name "Jogo (USA)"
	serial "SLUS-00001"
	rom ( name "Jogo (USA) (Track 01).bin" size 100 crc 0A0B0C0D md5 x )
	rom ( name "Jogo (USA) (Track 02).bin" size 50 crc DEADBEEF )
)

game (
	name "Arcade"
	rom ( name arcade.zip size 1 crc 00000001 )
)
`
	entries, err := thumbs.ParseDAT(strings.NewReader(dat))
	if err != nil || len(entries) != 2 {
		t.Fatalf("entradas: %+v %v", entries, err)
	}
	e := entries[0]
	if e.Name != "Jogo (USA)" || e.Serial != "SLUS-00001" || len(e.Roms) != 2 || e.Roms[0].CRC != 0x0A0B0C0D || e.Roms[0].Size != 100 {
		t.Fatalf("jogo: %+v", e)
	}
	if entries[1].Roms[0].Name != "arcade.zip" {
		t.Fatalf("nome de rom sem aspas: %+v", entries[1])
	}
}

func TestTitleKey(t *testing.T) {
	same := [][2]string{
		{"The Legend of Zelda (USA)", "Legend of Zelda, The"},
		{"Pokémon Gold (USA)", "Pokemon Gold"},
		{"Super Mario World (U) [!]", "Super Mario World (USA)"},
		{"Tom & Jerry", "Tom and Jerry"},
	}
	for _, p := range same {
		if thumbs.TitleKey(p[0]) != thumbs.TitleKey(p[1]) {
			t.Errorf("%q e %q deviam bater: %q vs %q", p[0], p[1], thumbs.TitleKey(p[0]), thumbs.TitleKey(p[1]))
		}
	}
	if thumbs.TitleKey("Mega Man X") == thumbs.TitleKey("Mega Man X2") {
		t.Error("X e X2 são jogos diferentes")
	}
}

func TestByTitlePrefersUSAAndRelease(t *testing.T) {
	ix := thumbs.NewIndex([]thumbs.Entry{
		{Name: "Jogo (Japan)"}, {Name: "Jogo (USA) (Beta)"}, {Name: "Jogo (Europe)"}, {Name: "Jogo (USA)"},
	})
	if got := ix.ByTitle("Jogo (U) [!]"); got != "Jogo (USA)" {
		t.Fatalf("escolheu %q", got)
	}
}

type env struct {
	fx    fixture.Env
	store *thumbs.Store
	games map[string]library.Game
	fake  *fixture.FakeThumbs
}

func setup(t *testing.T) env {
	t.Helper()
	fx := fixture.NewEnv(t.TempDir())
	fake := &fixture.FakeThumbs{}
	srv := httptest.NewServer(fake)
	t.Cleanup(srv.Close)
	store := &thumbs.Store{Cache: filepath.Join(t.TempDir(), "cache"), Images: srv.URL + "/thumbs", DB: srv.URL + "/metadat"}
	games := map[string]library.Game{}
	for _, g := range library.Scan(fx.Settings.LibraryDirs) {
		games[g.Title] = g
	}
	return env{fx, store, games, fake}
}

func TestFetchAllWays(t *testing.T) {
	e := setup(t)
	want := map[string]thumbs.Match{
		"Super Mario World":               {Name: "Super Mario World (USA)", How: "crc"},
		"Mega Man X":                      {Name: "Mega Man X (USA)", How: "title"},
		"Sonic The Hedgehog 2":            {Name: "Sonic The Hedgehog 2 (World) (Rev A)", How: "crc"},
		"Super Mario Bros. 3":             {Name: "Super Mario Bros. 3 (USA)", How: "crc"},
		"FIFA Soccer 07":                  {Name: "FIFA Soccer 07 (USA) (En,Es)", How: "serial"},
		"Lara Croft Tomb Raider - Legend": {Name: "Lara Croft Tomb Raider - Legend (USA) (En,Fr,De,Es,It)", How: "serial"},
		"Crash Bandicoot":                 {Name: "Crash Bandicoot (USA)", How: "title"},
		"mspacman":                        {Name: "Ms. Pac-Man", How: "zip"},
	}
	for title, w := range want {
		g, ok := e.games[title]
		if !ok {
			t.Fatalf("jogo %q não está no fixture", title)
		}
		m, got, err := e.store.Fetch(context.Background(), g)
		if err != nil {
			t.Fatalf("%s: %v", title, err)
		}
		if m != w {
			t.Errorf("%s: %+v, esperado %+v", title, m, w)
		}
		if title == "mspacman" && got != 2 {
			t.Errorf("Ms. Pac-Man sem tela de título no servidor: devia baixar 2 imagens, baixou %d", got)
		}
		if e.store.Image(g, "cover") == "" || !e.store.Fetched(g) {
			t.Errorf("%s: capa não ficou em cache", title)
		}
	}
	// N64 sem nada no banco: marca como tentado, sem imagem e sem erro
	sm64 := e.games["Super Mario 64"]
	if m, _, err := e.store.Fetch(context.Background(), sm64); err != nil || m.Name != "" || !e.store.Fetched(sm64) {
		t.Fatalf("sem correspondência: %+v %v", m, err)
	}
	// o DAT foi baixado uma vez só por console
	dats := 0
	for _, r := range e.fake.Requests {
		if strings.Contains(r, "Super Nintendo") && strings.HasPrefix(r, "/metadat/") {
			dats++
		}
	}
	if dats != 1 {
		t.Errorf("DAT do SNES baixado %d vezes", dats)
	}
}

func TestOffline(t *testing.T) {
	e := setup(t)
	e.store.DB = "http://127.0.0.1:1/metadat"
	_, _, err := e.store.Fetch(context.Background(), e.games["Super Mario World"])
	if err == nil || !strings.Contains(err.Error(), "sem conexão") {
		t.Fatalf("sem internet devia dar erro claro: %v", err)
	}
}

func TestProviderLocalCoverWins(t *testing.T) {
	e := setup(t)
	base := &meta.Local{ESDEDir: func() string { return e.fx.ESDE }}
	p := &thumbs.Provider{Base: base, Store: e.store}
	smw, sonic := e.games["Super Mario World"], e.games["Sonic The Hedgehog 2"]
	e.store.Fetch(context.Background(), smw)
	e.store.Fetch(context.Background(), sonic)
	info := p.Lookup(smw)
	if !strings.Contains(info.Cover, "downloaded_media") || info.Snap == "" || info.TitleShot == "" {
		t.Fatalf("capa do ES-DE tem prioridade, print e título vêm do libretro: %+v", info)
	}
	if got := p.Lookup(sonic).Cover; !strings.Contains(got, filepath.Join("genesis", "covers")) {
		t.Fatalf("capa da pasta covers/ tem prioridade: %s", got)
	}
	mmx := e.games["Mega Man X"]
	e.store.Fetch(context.Background(), mmx)
	if got := p.Lookup(mmx).Cover; !strings.HasPrefix(got, e.store.Cache) {
		t.Fatalf("sem capa local, usa a do libretro: %s", got)
	}
	if _, err := os.Stat(got(p, mmx)); err != nil {
		t.Fatal(err)
	}
}

func got(p *thumbs.Provider, g library.Game) string { return p.Lookup(g).Cover }
