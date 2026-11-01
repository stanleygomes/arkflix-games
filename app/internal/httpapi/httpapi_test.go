package httpapi_test

import (
	"bufio"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"
	"time"

	"github.com/stanleygomes/arkflix-games/app/internal/core"
	"github.com/stanleygomes/arkflix-games/app/internal/fixture"
	"github.com/stanleygomes/arkflix-games/app/internal/gamepad"
	"github.com/stanleygomes/arkflix-games/app/internal/httpapi"
	"github.com/stanleygomes/arkflix-games/app/internal/settings"
)

func server(t *testing.T) *httptest.Server {
	t.Helper()
	env := fixture.NewEnv(t.TempDir())
	store, _ := settings.Open(env.Data, env.Settings)
	b := httpapi.NewBroker()
	pads := gamepad.New(t.TempDir())
	svc, err := core.New(core.Options{Version: "t", Mode: "web", Store: store, Emitter: b, Pads: pads})
	if err != nil {
		t.Fatal(err)
	}
	assets := fstest.MapFS{"index.html": {Data: []byte("<h1>arkflix</h1>")}, "app.js": {Data: []byte("js")}}
	srv := httptest.NewServer(httpapi.Handler(svc, b, httpapi.Options{Assets: assets, Test: true, Pads: pads}))
	t.Cleanup(srv.Close)
	return srv
}

func post(t *testing.T, url, body string, header bool) (int, map[string]json.RawMessage) {
	t.Helper()
	req, _ := http.NewRequest(http.MethodPost, url, strings.NewReader(body))
	if header {
		req.Header.Set("X-Arkflix", "1")
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var out map[string]json.RawMessage
	json.NewDecoder(resp.Body).Decode(&out)
	return resp.StatusCode, out
}

func TestCall(t *testing.T) {
	srv := server(t)
	code, out := post(t, srv.URL+"/api/call/Bootstrap", "[]", true)
	if code != 200 {
		t.Fatalf("Bootstrap: %d %s", code, out["error"])
	}
	var boot core.Boot
	json.Unmarshal(out["result"], &boot)
	if len(boot.Games) != 9 {
		t.Fatalf("jogos: %d", len(boot.Games))
	}
	code, out = post(t, srv.URL+"/api/call/SetFavorite", `["`+boot.Games[0].ID+`", true]`, true)
	if code != 200 || !strings.Contains(string(out["result"]), `"favorite":true`) {
		t.Fatalf("SetFavorite: %d %s", code, out)
	}
}

func TestCallErrors(t *testing.T) {
	srv := server(t)
	cases := []struct {
		path, body string
		header     bool
		code       int
	}{
		{"/api/call/Bootstrap", "[]", false, 403},
		{"/api/call/NaoExiste", "[]", true, 404},
		{"/api/call/WaitReady", "[1]", true, 404},
		{"/api/call/SetFavorite", `["x"]`, true, 400},
		{"/api/call/SetFavorite", `{"x":1}`, true, 400},
		{"/api/call/SetFavorite", `[1, true]`, true, 400},
		{"/api/call/SetFavorite", `["nao-existe", true]`, true, 422},
	}
	for _, c := range cases {
		if code, out := post(t, srv.URL+c.path, c.body, c.header); code != c.code {
			t.Errorf("%s %s: %d (%s), esperado %d", c.path, c.body, code, out["error"], c.code)
		}
	}
}

func TestEventsAndPadInjection(t *testing.T) {
	srv := server(t)
	resp, err := http.Get(srv.URL + "/api/events")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	lines := make(chan string, 10)
	go func() {
		sc := bufio.NewScanner(resp.Body)
		for sc.Scan() {
			if strings.HasPrefix(sc.Text(), "data: ") {
				lines <- sc.Text()
			}
		}
	}()
	time.Sleep(100 * time.Millisecond)
	if code, _ := post(t, srv.URL+"/api/test/pad", `{"button":"south","pressed":true}`, false); code != 200 {
		t.Fatalf("inject: %d", code)
	}
	select {
	case l := <-lines:
		if !strings.Contains(l, `"name":"pad"`) || !strings.Contains(l, `"button":"south"`) {
			t.Fatalf("evento: %s", l)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("evento do controle não chegou pelo SSE")
	}
}

func TestStaticFallback(t *testing.T) {
	srv := server(t)
	for _, p := range []string{"/", "/configuracoes", "/app.js"} {
		resp, err := http.Get(srv.URL + p)
		if err != nil {
			t.Fatal(err)
		}
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		if resp.StatusCode != 200 || len(body) == 0 {
			t.Errorf("%s: %d", p, resp.StatusCode)
		}
	}
}
