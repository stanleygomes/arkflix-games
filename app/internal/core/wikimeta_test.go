package core_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stanleygomes/arkflix-games/app/internal/fixture"
	"github.com/stanleygomes/arkflix-games/app/internal/meta"
)

func fakeWiki(t *testing.T) *fixture.FakeWiki {
	fake := fixture.NewFakeWiki()
	srv := httptest.NewServer(fake)
	t.Cleanup(srv.Close)
	t.Setenv("ARKFLIX_WIKI_BASE", srv.URL)
	return fake
}

func TestFetchMeta(t *testing.T) {
	fake := fakeWiki(t)
	h := setup(t)
	r, err := h.svc.FetchMeta(h.game(t, "Sonic The Hedgehog 2").ID)
	if err != nil {
		t.Fatal(err)
	}
	if r.Year != 1992 || !strings.Contains(r.Summary, "Mega Drive") || r.Genres[0] != "Plataforma" || !strings.HasPrefix(r.Source, "https://pt.wikipedia.org/") {
		t.Fatalf("achado: %+v", r)
	}
	if ua, _ := fake.Agent.Load().(string); !strings.HasPrefix(ua, "arkflix-games/test (https://") {
		t.Fatalf("user-agent: %q", ua)
	}
	// buscar não grava nada: quem grava é o editor
	if g := h.game(t, "Sonic The Hedgehog 2"); g.Edited || g.Summary != "" {
		t.Fatalf("buscar gravou: %+v", g)
	}
}

func TestCompleteMetaOnlyEmptyFields(t *testing.T) {
	fakeWiki(t)
	h := setup(t)
	sonic, crash := h.game(t, "Sonic The Hedgehog 2"), h.game(t, "Crash Bandicoot")
	year, empty := 1991, ""
	if _, err := h.svc.SaveMeta(sonic.ID, meta.Override{Year: &year}); err != nil {
		t.Fatal(err)
	}
	if _, err := h.svc.SaveMeta(crash.ID, meta.Override{Summary: &empty}); err != nil {
		t.Fatal(err)
	}
	before := h.svc.MissingMeta()
	if before < 2 {
		t.Fatalf("faltando: %d", before)
	}
	if err := h.svc.CompleteMeta(); err != nil {
		t.Fatal(err)
	}
	if ev := h.rec.waitTask(t, "meta"); ev.State != "done" || !strings.HasPrefix(ev.Message, "2 jogo(s) completado(s)") {
		t.Fatalf("tarefa: %+v", ev)
	}
	sonic, crash = h.game(t, "Sonic The Hedgehog 2"), h.game(t, "Crash Bandicoot")
	if sonic.Year != 1991 || !strings.Contains(sonic.Summary, "Mega Drive") || len(sonic.Genres) != 1 {
		t.Fatalf("sonic (ano editado fica): %+v", sonic)
	}
	if crash.Summary != "" || crash.Year != 1996 || strings.Join(crash.Genres, ",") != "Plataforma,Tiro" {
		t.Fatalf("crash (descrição apagada na mão fica vazia): %+v", crash)
	}
	// não achados esperam: nada mais pra procurar agora
	if n := h.svc.MissingMeta(); n != 0 {
		t.Fatalf("depois de completar: %d", n)
	}
	if err := h.svc.CompleteMeta(); err == nil {
		t.Fatal("sem nada pra procurar devia avisar")
	}
}

func TestCompleteMetaCancel(t *testing.T) {
	// servidor que nunca responde: só o cancelamento tira a tarefa daí
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-r.Context().Done():
		case <-time.After(10 * time.Second):
		}
	}))
	t.Cleanup(srv.Close)
	t.Setenv("ARKFLIX_WIKI_BASE", srv.URL)
	h := setup(t)
	if err := h.svc.CompleteMeta(); err != nil {
		t.Fatal(err)
	}
	if err := h.svc.CompleteMeta(); err == nil {
		t.Fatal("duas de uma vez devia falhar")
	}
	time.Sleep(50 * time.Millisecond)
	if err := h.svc.CancelTask("meta"); err != nil {
		t.Fatal(err)
	}
	if ev := h.rec.waitTask(t, "meta"); ev.State != "cancelled" {
		t.Fatalf("cancelar: %+v", ev)
	}
	if h.game(t, "Crash Bandicoot").Edited {
		t.Fatal("cancelado não grava nada")
	}
}
