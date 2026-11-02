package wiki_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stanleygomes/arkflix-games/app/internal/fixture"
	"github.com/stanleygomes/arkflix-games/app/internal/wiki"
)

func client(t *testing.T) (*wiki.Client, *fixture.FakeWiki) {
	fake := fixture.NewFakeWiki()
	srv := httptest.NewServer(fake)
	t.Cleanup(srv.Close)
	return &wiki.Client{Base: srv.URL, UserAgent: "arkflix-games/test (teste)", Gate: &wiki.Gate{Gap: time.Millisecond}}, fake
}

func TestLookupPlatformAndPortuguese(t *testing.T) {
	c, fake := client(t)
	r, err := c.Lookup(context.Background(), "Sonic The Hedgehog 2 (World) (Rev A)", "genesis")
	if err != nil {
		t.Fatal(err)
	}
	if r.Entity != "Q35627" || r.Year != 1992 || !strings.Contains(r.Summary, "Mega Drive") {
		t.Fatalf("devia achar o do Mega Drive com resumo em pt: %+v", r)
	}
	if len(r.Genres) != 1 || r.Genres[0] != "Plataforma" || r.Source != "https://pt.wikipedia.org/wiki/Sonic_the_Hedgehog_2" {
		t.Fatalf("gênero/fonte: %+v", r)
	}
	if ua, _ := fake.Agent.Load().(string); ua != "arkflix-games/test (teste)" {
		t.Fatalf("user-agent: %q", ua)
	}
}

func TestLookupEnglishFallback(t *testing.T) {
	c, _ := client(t)
	r, err := c.Lookup(context.Background(), "Crash Bandicoot", "psx")
	if err != nil {
		t.Fatal(err)
	}
	if r.Year != 1996 || !strings.Contains(r.Summary, "Naughty Dog") || len(r.Genres) != 2 || r.Genres[1] != "Tiro" {
		t.Fatalf("crash: %+v", r)
	}
}

func TestLookupNotFound(t *testing.T) {
	c, _ := client(t)
	if _, err := c.Lookup(context.Background(), "Jogo Que Não Existe", "snes"); !errors.Is(err, wiki.ErrNotFound) {
		t.Fatalf("devia não achar: %v", err)
	}
}

func TestBusyAndGate(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer srv.Close()
	c := &wiki.Client{Base: srv.URL, Gate: &wiki.Gate{Gap: 80 * time.Millisecond}}
	start := time.Now()
	for range 2 {
		if _, err := c.Lookup(context.Background(), "x", ""); !errors.Is(err, wiki.ErrBusy) {
			t.Fatalf("429: %v", err)
		}
	}
	if time.Since(start) < 80*time.Millisecond {
		t.Fatal("o intervalo entre requisições não foi respeitado")
	}
}

func TestGenre(t *testing.T) {
	for in, want := range map[string]string{
		"jogo eletrônico de ação-aventura": "Ação-aventura",
		"metroidvania":                     "Metroidvania",
		"shooter game":                     "Shooter",
		"":                                 "",
	} {
		if got := wiki.Genre(in); got != want {
			t.Errorf("Genre(%q) = %q, want %q", in, got, want)
		}
	}
}
