package core_test

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stanleygomes/arkflix-games/app/internal/core"
	"github.com/stanleygomes/arkflix-games/app/internal/fixture"
	"github.com/stanleygomes/arkflix-games/app/internal/gamepad"
	"github.com/stanleygomes/arkflix-games/app/internal/hooks"
	"github.com/stanleygomes/arkflix-games/app/internal/settings"
)

type recorder struct {
	mu     sync.Mutex
	events []string
	data   []any
}

func (r *recorder) Emit(name string, data any) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.events = append(r.events, name)
	r.data = append(r.data, data)
}

func (r *recorder) waitGame(t *testing.T, state string) core.GameEvent {
	t.Helper()
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		r.mu.Lock()
		for i, e := range r.events {
			if ev, ok := r.data[i].(core.GameEvent); ok && e == "game" && ev.State == state {
				r.mu.Unlock()
				return ev
			}
		}
		r.mu.Unlock()
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("evento game %s não chegou", state)
	return core.GameEvent{}
}

type harness struct {
	env  fixture.Env
	svc  *core.Service
	rec  *recorder
	pads *gamepad.Manager
}

func setup(t *testing.T) harness {
	t.Helper()
	t.Setenv("ARKFLIX_FAKE_SLEEP", "0.05")
	env := fixture.NewEnv(t.TempDir())
	store, err := settings.Open(env.Data, env.Settings)
	if err != nil {
		t.Fatal(err)
	}
	rec := &recorder{}
	pads := gamepad.New(t.TempDir())
	// scripts de evento rodam em segundo plano: espera antes do TempDir ser apagado
	runner := &hooks.Runner{Dir: filepath.Join(env.Data, "scripts"), Log: filepath.Join(env.Data, "logs", "scripts.log")}
	t.Cleanup(runner.Wait)
	svc, err := core.New(core.Options{Version: "test", Mode: "web", Store: store, Emitter: rec, Pads: pads, Hooks: runner})
	if err != nil {
		t.Fatal(err)
	}
	return harness{env: env, svc: svc, rec: rec, pads: pads}
}

func (h harness) game(t *testing.T, name string) core.GameView {
	t.Helper()
	b, _ := h.svc.Bootstrap()
	for _, g := range b.Games {
		if g.Name == name || g.Title == name {
			return g
		}
	}
	t.Fatalf("jogo %q não está na biblioteca", name)
	return core.GameView{}
}

func TestBootstrap(t *testing.T) {
	h := setup(t)
	b, err := h.svc.Bootstrap()
	if err != nil {
		t.Fatal(err)
	}
	if len(b.Games) != 9 || len(b.Systems) != 10 || len(b.RetroPad) != 16 {
		t.Fatalf("boot: %d jogos, %d consoles, %d botões", len(b.Games), len(b.Systems), len(b.RetroPad))
	}
	if _, err := os.Stat(filepath.Join(h.env.Data, "retroarch-input.cfg")); err != nil {
		t.Fatal("config de entrada do RetroArch não foi gerada ao abrir")
	}
}

func TestFavorite(t *testing.T) {
	h := setup(t)
	g := h.game(t, "Super Mario World")
	v, err := h.svc.SetFavorite(g.ID, true)
	if err != nil || !v.Favorite {
		t.Fatalf("favoritar: %+v %v", v, err)
	}
	if !h.game(t, "Super Mario World").Favorite {
		t.Fatal("favorito não persistiu")
	}
	if _, err := h.svc.SetFavorite("nao-existe", true); err == nil {
		t.Fatal("jogo inexistente devia falhar")
	}
}

func TestPlayAndResume(t *testing.T) {
	h := setup(t)
	g := h.game(t, "Super Mario World")
	if err := h.svc.Play(g.ID, ""); err != nil {
		t.Fatal(err)
	}
	if err := h.svc.Play(g.ID, ""); err == nil || !strings.Contains(err.Error(), "já tem um jogo") {
		t.Fatalf("segundo jogo ao mesmo tempo devia ser recusado: %v", err)
	}
	ev := h.rec.waitGame(t, "exited")
	if ev.Error != "" || ev.ID != g.ID {
		t.Fatalf("saída: %+v", ev)
	}
	hookLog := filepath.Join(filepath.Dir(h.env.Home), "hook.log")
	deadline := time.Now().Add(5 * time.Second)
	for {
		data, _ := os.ReadFile(hookLog)
		if strings.Contains(string(data), "Super Mario World (U) [!].zip|Super Mario World|snes|") {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("script de fim de jogo não rodou com os argumentos certos: %q", data)
		}
		time.Sleep(20 * time.Millisecond)
	}
	if info := h.svc.Scripts(); len(info.Events["game-end"]) != 1 || info.Events["game-start"] == nil {
		t.Fatalf("scripts: %+v", info)
	}
	if h.game(t, "Super Mario World").LastPlayed == 0 {
		t.Fatal("último jogado não registrado")
	}
	calls := h.env.Calls()
	if !strings.Contains(calls, "retroarch|--fullscreen -L") || !strings.Contains(calls, "snes9x_libretro.so") || !strings.Contains(calls, "retroarch-input.cfg") {
		t.Fatalf("chamada do RetroArch: %s", calls)
	}

	list, err := h.svc.Saves(g.ID)
	if err != nil || len(list.States) != 2 {
		t.Fatalf("saves: %+v %v", list, err)
	}
	slot1 := list.States[1]
	h.rec = &recorder{}
	h2 := h
	if err := h2.svc.Play(g.ID, slot1.Token); err != nil {
		t.Fatal(err)
	}
	time.Sleep(400 * time.Millisecond)
	if !strings.Contains(h.env.Calls(), "--entryslot=1") {
		t.Fatalf("continuar do slot 1 devia passar --entryslot=1: %s", h.env.Calls())
	}
}

func TestPlayStateFromOtherGame(t *testing.T) {
	h := setup(t)
	smw := h.game(t, "Super Mario World")
	mmx := h.game(t, "Mega Man X")
	list, _ := h.svc.Saves(smw.ID)
	if err := h.svc.Play(mmx.ID, list.States[0].Token); err == nil {
		t.Fatal("state de outro jogo devia ser recusado")
	}
	if err := h.svc.Play(mmx.ID, "Li4vLi4vZXRjL3Bhc3N3ZA"); err == nil {
		t.Fatal("token forjado devia ser recusado")
	}
}

func TestPlayPSPWithState(t *testing.T) {
	h := setup(t)
	g := h.game(t, "Lara Croft Tomb Raider - Legend")
	list, _ := h.svc.Saves(g.ID)
	if len(list.States) != 1 {
		t.Fatalf("psp states: %+v", list)
	}
	if err := h.svc.Play(g.ID, list.States[0].Token); err != nil {
		t.Fatal(err)
	}
	h.rec.waitGame(t, "exited")
	if !strings.Contains(h.env.Calls(), "ppsspp|--fullscreen --state=") {
		t.Fatalf("ppsspp: %s", h.env.Calls())
	}
}

func TestDeleteState(t *testing.T) {
	h := setup(t)
	g := h.game(t, "Super Mario World")
	list, _ := h.svc.Saves(g.ID)
	after, err := h.svc.DeleteState(g.ID, list.States[0].Token)
	if err != nil || len(after.States) != 1 {
		t.Fatalf("apagar: %+v %v", after, err)
	}
}

func TestImport(t *testing.T) {
	h := setup(t)
	items := h.svc.AnalyzeImport([]string{
		filepath.Join(h.env.Inbox, "Tetris DX.gbc"),
		filepath.Join(h.env.Inbox, "Metal Slug X (USA).zip"),
		filepath.Join(h.env.Inbox, "Crazy Taxi.gdi"),
	})
	res, err := h.svc.Import(items)
	if err != nil {
		t.Fatal(err)
	}
	if !res[0].OK || !res[1].OK || res[2].OK {
		t.Fatalf("resultado: %+v", res)
	}
	h.game(t, "Tetris DX")
	h.game(t, "Metal Slug X")
}

func TestInputAndSettings(t *testing.T) {
	h := setup(t)
	m, _ := h.svc.GetInput()
	m.Pad["a"] = "south"
	res, err := h.svc.SaveInput(m)
	if err != nil || len(res.Conflicts) != 1 {
		t.Fatalf("conflito devia ser apontado: %+v %v", res, err)
	}
	cfg := h.svc.GetSettings()
	cfg.A11y.FontScale = 9
	cfg.A11y.HighContrast = true
	saved, err := h.svc.SaveSettings(cfg)
	if err != nil || saved.A11y.FontScale != 1 || !saved.A11y.HighContrast {
		t.Fatalf("settings: %+v %v", saved.A11y, err)
	}
}

func TestThumbMedia(t *testing.T) {
	h := setup(t)
	g := h.game(t, "FIFA Soccer 07")
	list, _ := h.svc.Saves(g.ID)
	rec := httptest.NewRecorder()
	core.Media(h.svc).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/media/thumb/"+list.States[0].Token, nil))
	if rec.Code != 200 || rec.Header().Get("Content-Type") != "image/png" {
		t.Fatalf("miniatura do p2s: %d %s", rec.Code, rec.Header().Get("Content-Type"))
	}
	rec = httptest.NewRecorder()
	core.Media(h.svc).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/media/thumb/Li4vZXRjL3Bhc3N3ZA", nil))
	if rec.Code != 404 {
		t.Fatal("token fora das pastas de save devia dar 404")
	}
}

func TestPadEventsReachUI(t *testing.T) {
	h := setup(t)
	h.pads.Inject(gamepad.Event{Button: "south", Pressed: true})
	h.rec.mu.Lock()
	defer h.rec.mu.Unlock()
	if len(h.rec.events) == 0 || h.rec.events[len(h.rec.events)-1] != "pad" {
		t.Fatalf("evento do controle não chegou: %v", h.rec.events)
	}
}

func TestLocalMetadata(t *testing.T) {
	h := setup(t)
	smw := h.game(t, "Super Mario World")
	if smw.Year != 1990 || smw.Cover == "" || !strings.Contains(smw.Summary, "Yoshi") {
		t.Fatalf("metadados do gamelist do ES-DE: %+v", smw)
	}
	rec := httptest.NewRecorder()
	core.Media(h.svc).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, smw.Cover, nil))
	if rec.Code != 200 || rec.Body.Len() == 0 {
		t.Fatalf("capa: %d", rec.Code)
	}
	sonic := h.game(t, "Sonic The Hedgehog 2")
	if sonic.Cover == "" {
		t.Fatal("capa da pasta covers/ ao lado da ROM não foi achada")
	}
	if crash := h.game(t, "Crash Bandicoot"); crash.Cover != "" || crash.Name != "Crash Bandicoot" {
		t.Fatalf("sem metadados, usa o título limpo e sem capa: %+v", crash)
	}
	rec = httptest.NewRecorder()
	core.Media(h.svc).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/media/cover/nao-existe", nil))
	if rec.Code != 404 {
		t.Fatal("capa de jogo inexistente devia dar 404")
	}
}
