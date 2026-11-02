package cheevos_test

import (
	"archive/zip"
	"context"
	"errors"
	"io"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"

	"github.com/stanleygomes/arkflix-games/app/internal/cheevos"
	"github.com/stanleygomes/arkflix-games/app/internal/fixture"
)

// newZip grava os arquivos na ordem dada (o ZipBytes do fixture usa mapa).
func newZip(w io.Writer, names []string, data [][]byte) *zip.Writer {
	zw := zip.NewWriter(w)
	for i, n := range names {
		f, _ := zw.Create(n)
		f.Write(data[i])
	}
	return zw
}

func TestClient(t *testing.T) {
	fake := fixture.NewFakeRA()
	srv := httptest.NewServer(fake)
	defer srv.Close()
	c := &cheevos.Client{Base: srv.URL, HTTP: srv.Client()}
	ctx := context.Background()

	if _, err := c.Login(ctx, "stanley", "errada"); !errors.Is(err, cheevos.ErrAuth) {
		t.Fatalf("senha errada: %v", err)
	}
	acc, err := c.Login(ctx, fixture.FakeRAUser, fixture.FakeRAPass)
	if err != nil || acc.Token != fixture.FakeRAToken || acc.User != "Stanley" {
		t.Fatalf("login: %+v %v", acc, err)
	}
	if id, err := c.GameID(ctx, strings.Repeat("0", 32)); err != nil || id != 0 {
		t.Fatalf("hash desconhecido: %d %v", id, err)
	}
	id, err := c.GameID(ctx, fixture.FakeRAHash)
	if err != nil || id != fixture.FakeRAGame {
		t.Fatalf("hash do SMW: %d %v", id, err)
	}
	g, err := c.Game(ctx, acc, id)
	if err != nil || g.Title != "Super Mario World" || len(g.Achievements) != 3 {
		t.Fatalf("conquistas (sem a não promovida): %+v %v", g, err)
	}
	if a := g.Achievements[0]; a.Points != 5 || !strings.HasSuffix(a.BadgeLocked, "/Badge/00101_lock.png") {
		t.Fatalf("conquista: %+v", a)
	}
	got, err := c.Unlocks(ctx, acc, id, false)
	if err != nil || !slices.Equal(got, []int{1001}) {
		t.Fatalf("desbloqueadas: %v %v", got, err)
	}
	if _, err := c.Unlocks(ctx, cheevos.Account{User: "stanley", Token: "velho"}, id, false); !errors.Is(err, cheevos.ErrToken) {
		t.Fatalf("token vencido: %v", err)
	}
	for _, r := range fake.Calls() {
		if !slices.Contains([]string{"login2", "gameid", "achievementsets", "unlocks"}, r) {
			t.Fatalf("chamada que mexe no perfil: %s", r)
		}
	}
}

func TestClientBadgeFallback(t *testing.T) {
	srv := httptest.NewServer(fakeNoBadgeURL{})
	defer srv.Close()
	c := &cheevos.Client{Base: srv.URL}
	g, err := c.Game(context.Background(), cheevos.Account{User: "u", Token: "t"}, 1)
	if err != nil || len(g.Achievements) != 1 {
		t.Fatalf("%+v %v", g, err)
	}
	if a := g.Achievements[0]; a.Badge != cheevos.MediaBase+"/Badge/00042.png" || a.BadgeLocked != cheevos.MediaBase+"/Badge/00042_lock.png" {
		t.Fatalf("imagem montada pelo BadgeName: %+v", a)
	}
}
