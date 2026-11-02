package core_test

import (
	"errors"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stanleygomes/arkflix-games/app/internal/core"
	"github.com/stanleygomes/arkflix-games/app/internal/fixture"
	"github.com/stanleygomes/arkflix-games/app/internal/settings"
	"github.com/stanleygomes/arkflix-games/app/internal/updater"
)

type updHarness struct {
	svc   *core.Service
	rec   *recorder
	cache string
	data  string
	exe   string
}

// newUpd monta o serviço com um GitHub falso; cache/data vazios = novos.
func newUpd(t *testing.T, api string, upd core.UpdateOptions, cache, data string) updHarness {
	t.Helper()
	if cache == "" {
		cache = t.TempDir()
	}
	if data == "" {
		data = t.TempDir()
	}
	env := fixture.NewEnv(t.TempDir())
	store, err := settings.Open(data, env.Settings)
	if err != nil {
		t.Fatal(err)
	}
	upd.API = api
	if upd.Current == "" {
		upd.Current = "v1.0.0"
	}
	rec := &recorder{}
	svc, err := core.New(core.Options{Version: "test", Mode: "web", Store: store, Emitter: rec, Cache: cache, Update: upd})
	if err != nil {
		t.Fatal(err)
	}
	core.SetUpdatePlatform(svc, "linux", "amd64")
	// prefixo de mentira: nunca o binário do teste
	exe := filepath.Join(t.TempDir(), "prefix", "bin", "arkflix-games")
	os.MkdirAll(filepath.Dir(exe), 0o755)
	os.WriteFile(exe, []byte("velho"), 0o755)
	core.SetUpdateInstaller(svc, func() (updater.Installer, error) {
		return updater.Installer{GOOS: "linux", Exe: exe}, nil
	})
	return updHarness{svc: svc, rec: rec, cache: cache, data: data, exe: exe}
}

func fakeGitHub(t *testing.T, f *fixture.FakeReleases) string {
	srv := httptest.NewServer(f)
	t.Cleanup(srv.Close)
	return srv.URL
}

func (r *recorder) updates() []core.UpdateInfo {
	r.mu.Lock()
	defer r.mu.Unlock()
	var out []core.UpdateInfo
	for i, e := range r.events {
		if info, ok := r.data[i].(core.UpdateInfo); ok && e == "update" {
			out = append(out, info)
		}
	}
	return out
}

func TestUpdateCheckDownloadApply(t *testing.T) {
	h := newUpd(t, fakeGitHub(t, &fixture.FakeReleases{Tag: "v2.0.0", Notes: "\n- novidade\n"}), core.UpdateOptions{}, "", "")
	if st := h.svc.UpdateStatus(); st.State != "idle" || st.Available || st.Current != "v1.0.0" {
		t.Fatalf("antes de olhar: %+v", st)
	}
	info, err := h.svc.CheckUpdate()
	if err != nil {
		t.Fatal(err)
	}
	if !info.Available || info.Latest != "v2.0.0" || info.Notes != "- novidade" ||
		info.Asset != "arkflix-games-linux-x86_64.tar.gz" || info.Size == 0 || !strings.HasSuffix(info.URL, "/releases/tag/v2.0.0") {
		t.Fatalf("checagem: %+v", info)
	}
	if ev := h.rec.updates(); len(ev) != 1 || !ev[0].Available {
		t.Fatalf("evento update: %+v", ev)
	}
	if err := h.svc.DownloadUpdate(); err != nil {
		t.Fatal(err)
	}
	if ev := h.rec.waitTask(t, "update"); ev.State != "done" {
		t.Fatalf("tarefa: %+v", ev)
	}
	if st := h.svc.UpdateStatus(); st.State != "ready" || st.Progress != 1 {
		t.Fatalf("depois do download: %+v", st)
	}
	if ev := h.rec.updates(); ev[len(ev)-1].State != "ready" {
		t.Fatalf("último evento: %+v", ev[len(ev)-1])
	}

	core.ApplyUpdateOnQuit(h.svc)
	if got, _ := os.ReadFile(h.exe); !strings.Contains(string(got), "v2.0.0") {
		t.Fatalf("binário não trocado: %q", got)
	}
	if _, err := os.Stat(filepath.Join(h.cache, "update")); err == nil {
		t.Fatal("pasta do download devia sumir depois de instalar")
	}
}

func TestUpdateChecksumMismatch(t *testing.T) {
	h := newUpd(t, fakeGitHub(t, &fixture.FakeReleases{Tag: "v2.0.0", BadSum: true}), core.UpdateOptions{}, "", "")
	if _, err := h.svc.CheckUpdate(); err != nil {
		t.Fatal(err)
	}
	h.svc.DownloadUpdate()
	if ev := h.rec.waitTask(t, "update"); ev.State != "error" || !strings.Contains(ev.Message, "checksum") {
		t.Fatalf("tarefa: %+v", ev)
	}
	st := h.svc.UpdateStatus()
	if st.State != "error" || st.Error == "" {
		t.Fatalf("estado: %+v", st)
	}
	files, _ := filepath.Glob(filepath.Join(h.cache, "update", "*"))
	if len(files) != 0 {
		t.Fatalf("arquivo recusado ficou: %v", files)
	}
	core.ApplyUpdateOnQuit(h.svc) // não está pronto: não instala
	if got, _ := os.ReadFile(h.exe); string(got) != "velho" {
		t.Fatal("instalou sem download conferido")
	}
}

func TestUpdateNotNewer(t *testing.T) {
	api := fakeGitHub(t, &fixture.FakeReleases{Tag: "v1.0.0"})
	for _, current := range []string{"v1.0.0", "v1.0.0-3-gabc-dirty", "dev", "v1.2.0"} {
		h := newUpd(t, api, core.UpdateOptions{Current: current}, "", "")
		info, err := h.svc.CheckUpdate()
		if err != nil || info.Available || info.State != "idle" {
			t.Fatalf("%s: %+v %v", current, info, err)
		}
		if err := h.svc.DownloadUpdate(); err == nil {
			t.Fatalf("%s: sem versão nova não baixa", current)
		}
		if len(h.rec.updates()) != 0 {
			t.Fatalf("%s: sem versão nova não avisa", current)
		}
	}
}

func TestUpdateReadySurvivesRestartAndDryRun(t *testing.T) {
	api := fakeGitHub(t, &fixture.FakeReleases{Tag: "v2.0.0"})
	h := newUpd(t, api, core.UpdateOptions{DryRun: true}, "", "")
	h.svc.CheckUpdate()
	h.svc.DownloadUpdate()
	h.rec.waitTask(t, "update")

	// reabre sem rede: o download pronto continua valendo
	again := newUpd(t, "http://127.0.0.1:1", core.UpdateOptions{DryRun: true}, h.cache, h.data)
	if st := again.svc.UpdateStatus(); st.State != "ready" || st.Latest != "v2.0.0" || !st.Available {
		t.Fatalf("depois de reabrir: %+v", st)
	}
	core.SetUpdateInstaller(again.svc, func() (updater.Installer, error) {
		return updater.Installer{}, errors.New("ambiente falso não chama o instalador")
	})
	core.ApplyUpdateOnQuit(again.svc)
	if st := again.svc.UpdateStatus(); st.State != "ready" {
		t.Fatalf("dry run mexeu no estado: %+v", st)
	}

	// já na versão baixada (instalou): a pasta é limpa
	done := newUpd(t, api, core.UpdateOptions{Current: "v2.0.0"}, h.cache, h.data)
	if st := done.svc.UpdateStatus(); st.State != "idle" {
		t.Fatalf("já atualizado: %+v", st)
	}
	if _, err := os.Stat(filepath.Join(h.cache, "update")); err == nil {
		t.Fatal("download velho devia sumir")
	}
}

func TestUpdateAutoCheckDownloads(t *testing.T) {
	core.SetCheckDelay(100 * time.Millisecond)
	t.Cleanup(func() { core.SetCheckDelay(5 * time.Second) })
	h := newUpd(t, fakeGitHub(t, &fixture.FakeReleases{}), core.UpdateOptions{Check: true, DryRun: true}, "", "")
	if ev := h.rec.waitTask(t, "update"); ev.State != "done" {
		t.Fatalf("download automático: %+v", ev)
	}
	if st := h.svc.UpdateStatus(); st.State != "ready" || st.Latest != "v99.0.0" {
		t.Fatalf("estado: %+v", st)
	}
}

func TestUpdateAutoCheckRespectsSetting(t *testing.T) {
	core.SetCheckDelay(100 * time.Millisecond)
	t.Cleanup(func() { core.SetCheckDelay(5 * time.Second) })
	data := t.TempDir()
	env := fixture.NewEnv(t.TempDir())
	store, _ := settings.Open(data, env.Settings)
	cfg := store.Settings()
	cfg.AutoUpdate = false
	store.Save(cfg)
	h := newUpd(t, fakeGitHub(t, &fixture.FakeReleases{}), core.UpdateOptions{Check: true, DryRun: true}, "", data)
	deadline := time.Now().Add(5 * time.Second)
	for len(h.rec.updates()) == 0 && time.Now().Before(deadline) {
		time.Sleep(20 * time.Millisecond)
	}
	time.Sleep(200 * time.Millisecond)
	ev := h.rec.updates()
	if len(ev) != 1 || !ev[0].Available || h.svc.UpdateStatus().State != "idle" {
		t.Fatalf("sem atualizar sozinho só avisa: %+v", ev)
	}
}

func TestAutoUpdateDefaultOn(t *testing.T) {
	if !settings.DefaultsFor("linux", "/home/x").AutoUpdate {
		t.Fatal("atualizar sozinho vem ligado")
	}
}

func TestUpdateLinuxReadOnlyFailsEarly(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root escreve em tudo")
	}
	h := newUpd(t, fakeGitHub(t, &fixture.FakeReleases{}), core.UpdateOptions{}, "", "")
	bin := filepath.Dir(h.exe)
	os.Chmod(bin, 0o555)
	t.Cleanup(func() { os.Chmod(bin, 0o755) })
	h.svc.CheckUpdate()
	err := h.svc.DownloadUpdate()
	if err == nil || !strings.Contains(err.Error(), "install.sh") {
		t.Fatalf("sem permissão devia mandar reinstalar: %v", err)
	}
	if st := h.svc.UpdateStatus(); st.State != "error" || st.Error == "" {
		t.Fatalf("estado: %+v", st)
	}
}

type startRec struct {
	mu    sync.Mutex
	start [][]string
}

func (r *startRec) Run(string, ...string) error { return nil }
func (r *startRec) Start(name string, args ...string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.start = append(r.start, append([]string{name}, args...))
	return nil
}

func (r *recorder) waitEvent(t *testing.T, name string) any {
	t.Helper()
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		r.mu.Lock()
		for i, e := range r.events {
			if e == name {
				d := r.data[i]
				r.mu.Unlock()
				return d
			}
		}
		r.mu.Unlock()
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("evento %s não chegou", name)
	return nil
}

func TestUpdateNowInstallsAndRestarts(t *testing.T) {
	core.SetRestartDelay(0)
	h := newUpd(t, fakeGitHub(t, &fixture.FakeReleases{Tag: "v2.0.0"}), core.UpdateOptions{}, "", "")
	run := &startRec{}
	core.SetUpdateInstaller(h.svc, func() (updater.Installer, error) {
		return updater.Installer{GOOS: "linux", Exe: h.exe, Run: run}, nil
	})
	if err := h.svc.UpdateNow(); err == nil {
		t.Fatal("sem versão nova devia recusar")
	}
	if _, err := h.svc.CheckUpdate(); err != nil {
		t.Fatal(err)
	}
	// um clique: baixa, instala e reabre
	if err := h.svc.UpdateNow(); err != nil {
		t.Fatal(err)
	}
	if tag := h.rec.waitEvent(t, "restart"); tag != "v2.0.0" {
		t.Fatalf("restart: %v", tag)
	}
	if got, _ := os.ReadFile(h.exe); !strings.Contains(string(got), "v2.0.0") {
		t.Fatalf("binário não trocado: %q", got)
	}
	run.mu.Lock()
	defer run.mu.Unlock()
	if len(run.start) != 1 || run.start[0][0] != "sh" || run.start[0][3] != h.exe {
		t.Fatalf("reabrir o app: %v", run.start)
	}
	if st := h.svc.UpdateStatus(); st.State != "restarting" || st.Checked == 0 {
		t.Fatalf("estado: %+v", st)
	}
	core.ApplyUpdateOnQuit(h.svc) // já instalou: não faz de novo (nem quebra)
}
