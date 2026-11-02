package emuinstall

import (
	"archive/zip"
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
)

func zipOf(t *testing.T, files map[string]string) []byte {
	t.Helper()
	var buf bytes.Buffer
	w := zip.NewWriter(&buf)
	for name, body := range files {
		f, err := w.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		f.Write([]byte(body))
	}
	w.Close()
	return buf.Bytes()
}

func testdata(t *testing.T, name string) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatal(err)
	}
	return data
}

// fakeWeb imita o buildbot, o ppsspp.org e a API do GitHub.
type fakeWeb struct {
	*httptest.Server
	hits  atomic.Int32
	mu    sync.Mutex
	paths []string
}

func newFakeWeb(t *testing.T) *fakeWeb {
	w := &fakeWeb{}
	files := map[string][]byte{
		"/bb/stable/": []byte(`<a href="/stable/1.9.0/">1.9.0</a> <a href="/stable/1.30.1/">1.30.1</a> <a href="/stable/1.22.2/">`),
		"/bb/stable/1.30.1/windows/x86_64/RetroArch.7z":             testdata(t, "retroarch.7z"),
		"/pp/1_20_4/ppsspp_win.zip":                                 zipOf(t, map[string]string{"PPSSPPWindows64.exe": "PSP", "assets/ui.ini": "ui"}),
		"/gh/repos/PCSX2/pcsx2/releases/latest":                     nil, // montado abaixo (precisa da URL)
		"/dl/pcsx2-v9-windows-x64-Qt.7z":                            testdata(t, "pcsx2.7z"),
		"/dl/pcsx2-v9-macos-Qt.tar.xz":                              []byte("PCSX2-v9.app"),
		"/bb/stable/1.30.1/apple/osx/universal/RetroArch_Metal.dmg": []byte("RetroArch.app"),
		"/pp/1_20_4/PPSSPP_macOS.dmg":                               []byte("PPSSPPSDL.app"),
	}
	w.Server = httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		w.hits.Add(1)
		w.mu.Lock()
		w.paths = append(w.paths, r.URL.Path)
		w.mu.Unlock()
		if r.URL.Path == "/gh/repos/PCSX2/pcsx2/releases/latest" {
			fmt.Fprintf(rw, `{"tag_name":"v9","assets":[
				{"name":"pcsx2-v9-windows-x64-Qt-symbols.7z","browser_download_url":"%[1]s/dl/nope"},
				{"name":"pcsx2-v9-windows-x64-Qt.7z","browser_download_url":"%[1]s/dl/pcsx2-v9-windows-x64-Qt.7z"},
				{"name":"pcsx2-v9-macos-Qt.tar.xz","browser_download_url":"%[1]s/dl/pcsx2-v9-macos-Qt.tar.xz"}]}`, w.URL)
			return
		}
		if core, ok := strings.CutPrefix(r.URL.Path, "/bb/nightly/"); ok {
			name := strings.TrimSuffix(filepath.Base(core), ".zip")
			rw.Write(zipOf(t, map[string]string{name: "core " + core}))
			return
		}
		data, ok := files[r.URL.Path]
		if !ok {
			http.NotFound(rw, r)
			return
		}
		rw.Write(data)
	}))
	t.Cleanup(w.Close)
	return w
}

func (w *fakeWeb) got(sub string) bool {
	w.mu.Lock()
	defer w.mu.Unlock()
	for _, p := range w.paths {
		if strings.Contains(p, sub) {
			return true
		}
	}
	return false
}

// fakeRun registra os comandos e imita flatpak, hdiutil, tar e ditto.
type fakeRun struct {
	mu      sync.Mutex
	calls   []string
	list    string // saída do flatpak list
	noFlat  bool
	failCmd string
}

func (f *fakeRun) run(_ context.Context, name string, args ...string) ([]byte, error) {
	f.mu.Lock()
	line := name + " " + strings.Join(args, " ")
	f.calls = append(f.calls, line)
	f.mu.Unlock()
	if f.failCmd != "" && strings.Contains(line, f.failCmd) {
		return nil, errors.New("boom")
	}
	switch name {
	case "flatpak":
		if f.noFlat {
			return nil, errors.New("não existe")
		}
		if len(args) > 0 && args[0] == "list" {
			return []byte(f.list), nil
		}
	case "hdiutil":
		if args[0] == "attach" {
			mnt := args[len(args)-2]
			app, _ := os.ReadFile(args[len(args)-1]) // o "dmg" falso é o nome do .app
			fakeApp(mnt, string(app), "1.30.1")
			os.Symlink("/Applications", filepath.Join(mnt, "Applications"))
		}
	case "tar":
		app, _ := os.ReadFile(args[1])
		fakeApp(args[len(args)-1], string(app), "9.0")
	case "ditto":
		return nil, os.CopyFS(args[1], os.DirFS(args[0]))
	}
	return nil, nil
}

func fakeApp(dir, app, version string) {
	exe := strings.TrimSuffix(app, ".app")
	if strings.HasPrefix(exe, "PCSX2") {
		exe = "PCSX2"
	}
	os.MkdirAll(filepath.Join(dir, app, "Contents", "MacOS"), 0o755)
	os.WriteFile(filepath.Join(dir, app, "Contents", "MacOS", exe), []byte("bin"), 0o755)
	os.WriteFile(filepath.Join(dir, app, "Contents", "Info.plist"), []byte(
		"<plist><dict><key>CFBundleShortVersionString</key>\n\t<string>"+version+"</string></dict></plist>"), 0o644)
}

func (f *fakeRun) has(sub string) bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, c := range f.calls {
		if strings.Contains(c, sub) {
			return true
		}
	}
	return false
}

func newInstaller(t *testing.T, goos string, web *fakeWeb, run *fakeRun) *Installer {
	home := t.TempDir()
	local := filepath.Join(home, "AppData", "Local")
	return &Installer{
		GOOS: goos, GOARCH: "amd64", Home: home, HTTP: web.Client(), Run: run.run,
		Env:      func(k string) string { return map[string]string{"LOCALAPPDATA": local}[k] },
		Buildbot: web.URL + "/bb", PPSSPP: web.URL + "/pp", GitHub: web.URL + "/gh", Flathub: "https://flathub.test/repo",
	}
}

// progresso: só sobe e termina em 1.
type progressLog struct {
	mu   sync.Mutex
	vals []float64
}

func (p *progressLog) add(v float64) {
	p.mu.Lock()
	p.vals = append(p.vals, v)
	p.mu.Unlock()
}

func (p *progressLog) check(t *testing.T) {
	t.Helper()
	for i, v := range p.vals {
		if v < 0 || v > 1 || (i > 0 && v < p.vals[i-1]-1e-9) {
			t.Fatalf("progresso fora de ordem: %v", p.vals)
		}
	}
	if len(p.vals) == 0 || p.vals[len(p.vals)-1] != 1 {
		t.Fatalf("progresso não terminou em 1: %v", p.vals)
	}
}

func TestWindows(t *testing.T) {
	web := newFakeWeb(t)
	in := newInstaller(t, "windows", web, &fakeRun{})
	st := in.Status(context.Background())
	if !st.CanInstall || len(st.Emulators) != 3 || len(st.Cores) != 8 || st.Emulators[0].Installed {
		t.Fatalf("antes: %+v", st)
	}
	var p progressLog
	msg, err := in.Install(context.Background(), nil, p.add)
	if err != nil || msg != "RetroArch, PPSSPP, PCSX2 e 8 cores prontos" {
		t.Fatalf("install: %q %v", msg, err)
	}
	p.check(t)
	ra := filepath.Join(in.env("LOCALAPPDATA"), "Programs", "RetroArch")
	for _, f := range []string{
		filepath.Join(ra, "retroarch.exe"), filepath.Join(ra, "assets", "menu.txt"),
		filepath.Join(ra, "cores", "snes9x_libretro.dll"), filepath.Join(ra, "cores", "melondsds_libretro.dll"),
		filepath.Join(in.env("LOCALAPPDATA"), "Programs", "PPSSPP", "PPSSPPWindows64.exe"),
		filepath.Join(in.env("LOCALAPPDATA"), "Programs", "PCSX2", "pcsx2-qt.exe"),
	} {
		if !isFile(f) {
			t.Fatalf("faltou %s", f)
		}
	}
	if !web.got("/bb/stable/1.30.1/") || !web.got("/nightly/windows/x86_64/latest/fbneo_libretro.dll.zip") || web.got("/dl/nope") {
		t.Fatalf("urls: %v", web.paths)
	}
	st = in.Status(context.Background())
	for _, e := range st.Emulators {
		if !e.Installed {
			t.Fatalf("depois: %+v", e)
		}
	}
	for _, c := range st.Cores {
		if !c.Installed {
			t.Fatalf("core: %+v", c)
		}
	}
	// atualizar mantém a config da instalação portátil
	cfg := filepath.Join(ra, "retroarch.cfg")
	os.WriteFile(cfg, []byte("meu"), 0o644)
	if msg, err := in.Install(context.Background(), []string{"retroarch"}, func(float64) {}); err != nil || msg != "RetroArch pronto" {
		t.Fatalf("update: %q %v", msg, err)
	}
	if data, _ := os.ReadFile(cfg); string(data) != "meu" {
		t.Fatal("update apagou a config")
	}
	// nada faltando = nada baixado
	before := web.hits.Load()
	if msg, err := in.Install(context.Background(), nil, func(float64) {}); err != nil || msg != "emuladores e cores já instalados" || web.hits.Load() != before {
		t.Fatalf("nada a fazer: %q %v", msg, err)
	}
}

func TestLinux(t *testing.T) {
	web := newFakeWeb(t)
	run := &fakeRun{list: "org.libretro.RetroArch\t1.22.2\tuser\nnet.pcsx2.PCSX2\tv2.8.2\tsystem\n"}
	in := newInstaller(t, "linux", web, run)
	st := in.Status(context.Background())
	if !st.CanInstall || !st.Emulators[0].Installed || st.Emulators[0].Version != "1.22.2" || st.Emulators[1].Installed {
		t.Fatalf("status: %+v", st)
	}
	var p progressLog
	msg, err := in.Install(context.Background(), nil, p.add)
	if err != nil || msg != "PPSSPP e 8 cores prontos" {
		t.Fatalf("install: %q %v", msg, err)
	}
	p.check(t)
	for _, c := range []string{
		"flatpak remote-add --user --if-not-exists flathub https://flathub.test/repo",
		"flatpak install --user -y --noninteractive flathub org.ppsspp.PPSSPP",
	} {
		if !run.has(c) {
			t.Fatalf("faltou %q: %v", c, run.calls)
		}
	}
	if run.has("install --user -y --noninteractive flathub org.libretro.RetroArch") {
		t.Fatal("instalou o que já tinha")
	}
	core := filepath.Join(in.Home, ".var", "app", "org.libretro.RetroArch", "config", "retroarch", "cores", "mesen_libretro.so")
	if data, _ := os.ReadFile(core); !strings.Contains(string(data), "linux/x86_64/latest/mesen_libretro.so.zip") {
		t.Fatalf("core: %q", data)
	}
	// atualizar: o do usuário com --user, o do sistema sem; PCSX2 libera a home
	if _, err := in.Install(context.Background(), []string{"retroarch", "pcsx2"}, func(float64) {}); err != nil {
		t.Fatal(err)
	}
	for _, c := range []string{
		"flatpak update --user -y --noninteractive org.libretro.RetroArch",
		"flatpak update -y --noninteractive net.pcsx2.PCSX2",
		"flatpak override --user --filesystem=host:ro net.pcsx2.PCSX2",
	} {
		if !run.has(c) {
			t.Fatalf("faltou %q: %v", c, run.calls)
		}
	}
	// uma falha não para o resto e aparece na mensagem
	run.failCmd = "org.ppsspp.PPSSPP"
	_, err = in.Install(context.Background(), []string{"ppsspp", "cores"}, func(float64) {})
	if err == nil || !strings.Contains(err.Error(), "8 cores prontos; falhou: PPSSPP") {
		t.Fatalf("falha parcial: %v", err)
	}
}

func TestMac(t *testing.T) {
	web := newFakeWeb(t)
	run := &fakeRun{}
	in := newInstaller(t, "darwin", web, run)
	in.GOARCH = "arm64"
	if msg, err := in.Install(context.Background(), []string{"retroarch", "pcsx2", "cores"}, func(float64) {}); err != nil || msg != "RetroArch, PCSX2 e 8 cores prontos" {
		t.Fatalf("install: %q %v", msg, err)
	}
	if _, err := in.Install(context.Background(), []string{"ppsspp"}, func(float64) {}); err != nil {
		t.Fatal(err)
	}
	st := in.Status(context.Background())
	want := map[string]string{"retroarch": "RetroArch.app", "ppsspp": "PPSSPPSDL.app", "pcsx2": "PCSX2.app"}
	for _, e := range st.Emulators {
		if !e.Installed || !strings.HasPrefix(e.Path, filepath.Join(in.Home, "Applications", want[e.ID])) || e.Version == "" {
			t.Fatalf("mac: %+v", e)
		}
	}
	if !run.has("hdiutil attach -nobrowse -readonly") || !run.has("hdiutil detach -force") || !run.has("tar -xf") {
		t.Fatalf("comandos: %v", run.calls)
	}
	if !web.got("/nightly/apple/osx/arm64/latest/snes9x_libretro.dylib.zip") || !web.got("RetroArch_Metal.dmg") {
		t.Fatalf("urls: %v", web.paths)
	}
	if !isFile(filepath.Join(in.Home, "Library", "Application Support", "RetroArch", "cores", "fbneo_libretro.dylib")) {
		t.Fatal("core do mac")
	}
	// reinstalar troca o .app inteiro, sem sobras
	if _, err := in.Install(context.Background(), []string{"pcsx2"}, func(float64) {}); err != nil {
		t.Fatal(err)
	}
	items, _ := os.ReadDir(filepath.Join(in.Home, "Applications"))
	if len(items) != 3 {
		t.Fatalf("sobras em ~/Applications: %v", items)
	}
}

func TestUnsupported(t *testing.T) {
	web := newFakeWeb(t)
	in := newInstaller(t, "linux", web, &fakeRun{noFlat: true})
	if st := in.Status(context.Background()); st.CanInstall || !strings.Contains(st.Reason, "Flatpak") {
		t.Fatalf("sem flatpak: %+v", st)
	}
	if _, err := in.Install(context.Background(), nil, func(float64) {}); err == nil {
		t.Fatal("sem flatpak devia falhar")
	}
	in = newInstaller(t, "windows", web, &fakeRun{})
	in.GOARCH = "arm64"
	if in.Status(context.Background()).CanInstall {
		t.Fatal("windows arm64")
	}
	if err := CheckNames([]string{"dolphin"}); err == nil {
		t.Fatal("nome desconhecido")
	}
}

func TestExtractRejectsEscape(t *testing.T) {
	for _, name := range []string{"../evil.exe", "a/../../evil.exe", "/etc/evil.exe", `..\evil.exe`, `C:\evil.exe`} {
		dir := t.TempDir()
		zipFile := filepath.Join(dir, "x.zip")
		os.WriteFile(zipFile, zipOf(t, map[string]string{"app.exe": "ok", name: "mal"}), 0o644)
		dst := filepath.Join(dir, "dst")
		if err := extractTree(zipFile, "app.exe", dst); err == nil || !strings.Contains(err.Error(), "perigoso") {
			t.Fatalf("%s: %v", name, err)
		}
		if isFile(filepath.Join(dir, "evil.exe")) {
			t.Fatalf("%s escapou", name)
		}
	}
}

// zip que declara menos bytes do que tem não passa do declarado.
func TestExtractRejectsLyingSize(t *testing.T) {
	var buf bytes.Buffer
	w := zip.NewWriter(&buf)
	f, _ := w.CreateRaw(&zip.FileHeader{Name: "app.exe", Method: zip.Store, CompressedSize64: 10, UncompressedSize64: 5})
	f.Write([]byte("0123456789"))
	w.Close()
	dir := t.TempDir()
	file := filepath.Join(dir, "x.zip")
	os.WriteFile(file, buf.Bytes(), 0o644)
	if err := extractTree(file, "app.exe", filepath.Join(dir, "dst")); err == nil {
		t.Fatal("tamanho mentiroso devia falhar")
	}
	if isFile(filepath.Join(dir, "dst", "app.exe")) {
		t.Fatal("arquivo pela metade ficou")
	}
}

func TestDownloadErrors(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/big" {
			w.Header().Set("Content-Length", "2000")
			w.Write(make([]byte, 2000))
			return
		}
		http.NotFound(w, r)
	}))
	defer srv.Close()
	in := &Installer{HTTP: srv.Client()}
	if _, err := in.download(context.Background(), srv.URL+"/x", t.TempDir(), 100, nil); err == nil || !strings.Contains(err.Error(), "404") {
		t.Fatalf("404: %v", err)
	}
	dir := t.TempDir()
	if _, err := in.download(context.Background(), srv.URL+"/big", dir, 100, nil); err == nil || !strings.Contains(err.Error(), "grande") {
		t.Fatalf("grande: %v", err)
	}
	if items, _ := os.ReadDir(dir); len(items) != 0 {
		t.Fatal("sobrou temporário")
	}
	// listagem fora do ar: usa a versão fixa
	in.Buildbot = srv.URL + "/fora"
	if v := in.retroArchVersion(context.Background()); v != RetroArchStable {
		t.Fatal(v)
	}
}

func TestSandbox(t *testing.T) {
	dir := t.TempDir()
	s := &Sandbox{Bin: filepath.Join(dir, "bin"), Cores: filepath.Join(dir, "cores"), Delay: 3}
	os.MkdirAll(s.Bin, 0o755)
	os.WriteFile(filepath.Join(s.Bin, "retroarch"), []byte("meu"), 0o755)
	var p progressLog
	msg, err := s.Install(context.Background(), nil, p.add)
	if err != nil || msg != "PPSSPP, PCSX2 e 8 cores prontos" {
		t.Fatalf("%q %v", msg, err)
	}
	p.check(t)
	if data, _ := os.ReadFile(filepath.Join(s.Bin, "retroarch")); string(data) != "meu" {
		t.Fatal("sandbox pisou no lançador existente")
	}
	t.Setenv("ARKFLIX_FAKE_INSTALL_FAIL", "pcsx2")
	if _, err := s.Install(context.Background(), []string{"pcsx2"}, func(float64) {}); err == nil {
		t.Fatal("falha simulada")
	}
}
