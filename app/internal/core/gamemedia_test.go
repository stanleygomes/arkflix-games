package core_test

import (
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stanleygomes/arkflix-games/app/internal/core"
)

func TestGameMediaAndNotes(t *testing.T) {
	h := setup(t)
	g := h.game(t, "Super Mario World")
	// a config do guia aponta capturas e gravações pra pasta do perfil
	if err := h.svc.Play(g.ID, ""); err != nil {
		t.Fatal(err)
	}
	h.rec.waitGame(t, "exited")
	cfg, err := os.ReadFile(filepath.Join(h.env.Data, "retroarch-guide.cfg"))
	if err != nil || !strings.Contains(string(cfg), "screenshot_directory") {
		t.Fatalf("config do guia: %q %v", cfg, err)
	}
	shots := filepath.Join(h.env.Data, "media", "screenshots")
	videos := filepath.Join(h.env.Data, "media", "videos")
	os.WriteFile(filepath.Join(shots, "Super Mario World (U) [!]-251102-101010.png"), []byte("png"), 0o644)
	os.WriteFile(filepath.Join(videos, "Super Mario World (U) [!]-2025_11_02.mkv"), []byte("mkv"), 0o644)
	os.WriteFile(filepath.Join(shots, "Outro Jogo-251102-101010.png"), []byte("x"), 0o644)
	later := time.Now().Add(time.Minute)
	os.Chtimes(filepath.Join(videos, "Super Mario World (U) [!]-2025_11_02.mkv"), later, later)

	list, err := h.svc.GameMedia(g.ID)
	if err != nil || len(list) != 2 || list[0].Kind != "video" || list[1].Kind != "image" {
		t.Fatalf("mídia: %+v %v", list, err)
	}
	rec := httptest.NewRecorder()
	core.Media(h.svc).ServeHTTP(rec, httptest.NewRequest("GET", "/media/shot/"+list[1].ID, nil))
	if rec.Code != 200 || rec.Body.String() != "png" {
		t.Fatalf("servir captura: %d %q", rec.Code, rec.Body.String())
	}
	// fora da pasta de mídia: não serve
	outside := httptest.NewRecorder()
	bad := core.EncodeToken(filepath.Join(h.env.Data, "settings.json"))
	core.Media(h.svc).ServeHTTP(outside, httptest.NewRequest("GET", "/media/shot/"+bad, nil))
	if outside.Code != 404 {
		t.Fatalf("serviu arquivo de fora: %d", outside.Code)
	}

	if err := h.svc.SetGameNote(g.ID, "  a chave tá no castelo 3 "); err != nil {
		t.Fatal(err)
	}
	if got := h.svc.GameNote(g.ID); got != "a chave tá no castelo 3" {
		t.Fatalf("anotação: %q", got)
	}
	if err := h.svc.SetGameNote(g.ID, strings.Repeat("x", 2001)); err == nil {
		t.Fatal("anotação gigante devia ser recusada")
	}
	h.svc.SetGameNote(g.ID, "")
	if got := h.svc.GameNote(g.ID); got != "" {
		t.Fatalf("anotação apagada: %q", got)
	}
}

func TestSwitchGame(t *testing.T) {
	h := setup(t)
	t.Setenv("ARKFLIX_FAKE_SLEEP", "30")
	core.SetQuitGrace(100 * time.Millisecond)
	smw := h.game(t, "Super Mario World")
	mmx := h.game(t, "Mega Man X")
	if err := h.svc.Play(smw.ID, ""); err != nil {
		t.Fatal(err)
	}
	h.rec.waitGame(t, "running")
	if g := h.svc.Guide(); g.Game != smw.ID || g.Started == 0 {
		t.Fatalf("guia: %+v", g)
	}
	if err := h.svc.SwitchGame(mmx.ID); err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(10 * time.Second)
	for h.svc.Guide().Game != mmx.ID {
		if time.Now().After(deadline) {
			t.Fatalf("não trocou: %+v", h.svc.Guide())
		}
		time.Sleep(50 * time.Millisecond)
	}
	h.svc.GameCommand("quit")
}
