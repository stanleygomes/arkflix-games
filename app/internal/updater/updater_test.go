package updater

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNewer(t *testing.T) {
	cases := []struct {
		latest, current string
		want            bool
	}{
		{"v1.9.4", "v1.9.3", true},
		{"v1.10.0", "v1.9.9", true},
		{"v2.0.0", "v1.99.99", true},
		{"v1.9.3", "v1.9.3", false},
		{"v1.9.2", "v1.9.3", false},
		{"v1.9.3", "v1.9.3-7-gabc", false},       // dev em cima da tag: não "volta"
		{"v1.9.3", "v1.9.3-7-gabc-dirty", false}, // idem, sujo
		{"v1.9.4", "v1.9.3-7-gabc-dirty", true},  // tag estritamente maior
		{"v1.9.4", "dev", false},
		{"v1.9.4", "abc1234", false},
		{"lixo", "v1.0.0", false},
		{"1.2.0", "v1.1.0", true},
	}
	for _, c := range cases {
		if got := Newer(c.latest, c.current); got != c.want {
			t.Errorf("Newer(%q, %q) = %v, esperado %v", c.latest, c.current, got, c.want)
		}
	}
}

func TestAssetName(t *testing.T) {
	if AssetName("linux", "amd64") != "arkflix-games-linux-x86_64.tar.gz" ||
		AssetName("windows", "amd64") != "arkflix-games-windows-x86_64-setup.exe" ||
		AssetName("darwin", "arm64") != "arkflix-games-macos-universal.dmg" ||
		AssetName("linux", "arm64") != "" {
		t.Fatal("nomes dos anexos não batem com o bin/package.sh")
	}
}

// server: GitHub falso com um anexo e o .sha256 (sum vazio = o certo).
func server(t *testing.T, data []byte, sum string) *httptest.Server {
	if sum == "" {
		h := sha256.Sum256(data)
		sum = hex.EncodeToString(h[:])
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/repos/o/r/releases/latest", func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("User-Agent") == "" {
			http.Error(w, "sem user agent", 403)
			return
		}
		base := "http://" + r.Host
		fmt.Fprintf(w, `{"tag_name":"v2.0.0","body":"  notas \n","html_url":"%s/rel","assets":[
			{"name":"a.tar.gz","size":%d,"browser_download_url":"%s/a.tar.gz"},
			{"name":"a.tar.gz.sha256","size":80,"browser_download_url":"%s/a.tar.gz.sha256"}]}`, base, len(data), base, base)
	})
	mux.HandleFunc("/a.tar.gz", func(w http.ResponseWriter, r *http.Request) { w.Write(data) })
	mux.HandleFunc("/a.tar.gz.sha256", func(w http.ResponseWriter, r *http.Request) { fmt.Fprintf(w, "%s  a.tar.gz\n", sum) })
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv
}

func TestLatestAndDownload(t *testing.T) {
	data := bytes.Repeat([]byte("x"), 100_000)
	srv := server(t, data, "")
	c := Client{API: srv.URL, Repo: "o/r"}
	ctx := context.Background()
	rel, err := c.Latest(ctx)
	if err != nil || rel.Tag != "v2.0.0" || rel.Body != "notas" {
		t.Fatalf("release: %+v %v", rel, err)
	}
	a, ok := rel.Find("a.tar.gz")
	s, okS := rel.Find("a.tar.gz.sha256")
	if !ok || !okS {
		t.Fatal("anexos")
	}
	sum, err := c.Sum(ctx, s.URL)
	if err != nil {
		t.Fatal(err)
	}
	dst := filepath.Join(t.TempDir(), "u", "a.tar.gz")
	var last float64
	if err := c.Download(ctx, a.URL, sum, dst, a.Size, func(p float64) { last = p }); err != nil {
		t.Fatal(err)
	}
	if got, _ := os.ReadFile(dst); !bytes.Equal(got, data) || last != 1 {
		t.Fatalf("arquivo ou progresso errado (%v)", last)
	}
}

func TestDownloadRejectsBadChecksum(t *testing.T) {
	srv := server(t, []byte("conteúdo"), strings.Repeat("ab", 32))
	c := Client{API: srv.URL, Repo: "o/r"}
	ctx := context.Background()
	sum, err := c.Sum(ctx, srv.URL+"/a.tar.gz.sha256")
	if err != nil {
		t.Fatal(err)
	}
	dst := filepath.Join(t.TempDir(), "a.tar.gz")
	err = c.Download(ctx, srv.URL+"/a.tar.gz", sum, dst, 0, nil)
	if err == nil || !strings.Contains(err.Error(), "checksum não confere") {
		t.Fatalf("devia recusar: %v", err)
	}
	if _, err := os.Stat(dst); err == nil {
		t.Fatal("arquivo recusado ficou no destino")
	}
	if _, err := os.Stat(dst + ".part"); err == nil {
		t.Fatal(".part ficou pra trás")
	}
}

type entry struct {
	name, body string
	typ        byte
}

func tarball(t *testing.T, entries ...entry) []byte {
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	for _, e := range entries {
		h := &tar.Header{Name: e.name, Mode: 0o755, Size: int64(len(e.body)), Typeflag: e.typ}
		if e.typ == tar.TypeSymlink {
			h.Linkname, h.Size = "/etc/passwd", 0
		}
		if err := tw.WriteHeader(h); err != nil {
			t.Fatal(err)
		}
		tw.Write([]byte(e.body))
	}
	tw.Close()
	gz.Close()
	return buf.Bytes()
}

func TestExtract(t *testing.T) {
	prefix := filepath.Join(t.TempDir(), "prefix")
	os.MkdirAll(filepath.Join(prefix, "bin"), 0o755)
	os.WriteFile(filepath.Join(prefix, "bin", "arkflix-games"), []byte("velho"), 0o755)
	tgz := tarball(t, entry{"prefix/", "", tar.TypeDir}, entry{"prefix/bin/arkflix-games", "novo", tar.TypeReg})
	if err := Extract(bytes.NewReader(tgz), prefix); err != nil {
		t.Fatal(err)
	}
	bin := filepath.Join(prefix, "bin", "arkflix-games")
	got, _ := os.ReadFile(bin)
	info, _ := os.Stat(bin)
	if string(got) != "novo" || info.Mode().Perm() != 0o755 {
		t.Fatalf("binário: %q %v", got, info.Mode())
	}
	if left, _ := filepath.Glob(filepath.Join(prefix, "bin", ".*")); len(left) > 0 {
		t.Fatalf("temporários ficaram: %v", left)
	}
}

func TestExtractRejectsTraversal(t *testing.T) {
	root := t.TempDir()
	prefix := filepath.Join(root, "prefix")
	os.MkdirAll(prefix, 0o755)
	bad := [][]entry{
		{{"prefix/../../fora", "x", tar.TypeReg}},
		{{"../fora", "x", tar.TypeReg}},
		{{"/etc/fora", "x", tar.TypeReg}},
		{{"prefix/bin/link", "", tar.TypeSymlink}},
		// o arquivo bom antes do ruim não pode ficar meio instalado
		{{"prefix/bin/ok", "x", tar.TypeReg}, {"prefix/../../fora", "x", tar.TypeReg}},
	}
	for _, entries := range bad {
		if err := Extract(bytes.NewReader(tarball(t, entries...)), prefix); err == nil {
			t.Errorf("devia recusar %v", entries)
		}
	}
	if _, err := os.Stat(filepath.Join(root, "fora")); err == nil {
		t.Fatal("escapou da pasta")
	}
	if _, err := os.Stat(filepath.Join(prefix, "bin", "ok")); err == nil {
		t.Fatal("pacote recusado deixou arquivo")
	}
	if err := Extract(strings.NewReader("não é gzip"), prefix); err == nil {
		t.Fatal("lixo devia falhar")
	}
}

func TestApplyLinux(t *testing.T) {
	prefix := filepath.Join(t.TempDir(), "build", "prefix")
	exe := filepath.Join(prefix, "bin", "arkflix-games")
	os.MkdirAll(filepath.Dir(exe), 0o755)
	os.WriteFile(exe, []byte("velho"), 0o755)
	pkg := filepath.Join(t.TempDir(), "a.tar.gz")
	os.WriteFile(pkg, tarball(t, entry{"prefix/bin/arkflix-games", "novo", tar.TypeReg}), 0o644)
	in := Installer{GOOS: "linux", Exe: exe}
	if err := in.Check(); err != nil {
		t.Fatal(err)
	}
	if err := in.Apply(pkg); err != nil {
		t.Fatal(err)
	}
	if got, _ := os.ReadFile(exe); string(got) != "novo" {
		t.Fatalf("binário não trocado: %q", got)
	}
	// fora de <prefix>/bin: não mexe, manda reinstalar
	odd := Installer{GOOS: "linux", Exe: filepath.Join(t.TempDir(), "arkflix-games")}
	if err := odd.Check(); err == nil || !strings.Contains(err.Error(), "install.sh") {
		t.Fatalf("layout estranho: %v", err)
	}
}

func TestApplyLinuxReadOnly(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root escreve em tudo")
	}
	prefix := t.TempDir()
	bin := filepath.Join(prefix, "bin")
	os.MkdirAll(bin, 0o755)
	os.Chmod(bin, 0o555)
	t.Cleanup(func() { os.Chmod(bin, 0o755) })
	in := Installer{GOOS: "linux", Exe: filepath.Join(bin, "arkflix-games")}
	if err := in.Check(); err == nil || !strings.Contains(err.Error(), "sem permissão") {
		t.Fatalf("pasta só leitura: %v", err)
	}
}

type fakeRunner struct {
	calls  []string
	runErr map[string]error
	ditto  func(dst string)
}

func (f *fakeRunner) Run(name string, args ...string) error {
	f.calls = append(f.calls, "run "+name+" "+strings.Join(args, " "))
	if name == "ditto" && f.ditto != nil {
		f.ditto(args[len(args)-1])
	}
	return f.runErr[name]
}

func (f *fakeRunner) Start(name string, args ...string) error {
	f.calls = append(f.calls, "start "+name+" "+strings.Join(args, " "))
	return nil
}

func TestApplyWindows(t *testing.T) {
	r := &fakeRunner{}
	in := Installer{GOOS: "windows", Exe: `C:\x\arkflix-games.exe`, Run: r}
	if err := in.Apply(`C:\Users\o'neil\cache\setup.exe`); err != nil {
		t.Fatal(err)
	}
	if len(r.calls) != 1 || !strings.HasPrefix(r.calls[0], "start powershell") ||
		!strings.Contains(r.calls[0], `'C:\Users\o''neil\cache\setup.exe' -ArgumentList '/S'`) {
		t.Fatalf("comando: %v", r.calls)
	}
}

func macApp(t *testing.T) (app, exe string) {
	app = filepath.Join(t.TempDir(), "Applications", "arkflix-games.app")
	exe = filepath.Join(app, "Contents", "MacOS", "arkflix-games")
	os.MkdirAll(filepath.Dir(exe), 0o755)
	os.WriteFile(exe, []byte("velho"), 0o755)
	return app, exe
}

func TestApplyDarwin(t *testing.T) {
	app, exe := macApp(t)
	r := &fakeRunner{ditto: func(dst string) {
		os.MkdirAll(filepath.Join(dst, "Contents", "MacOS"), 0o755)
		os.WriteFile(filepath.Join(dst, "Contents", "MacOS", "arkflix-games"), []byte("novo"), 0o755)
	}}
	in := Installer{GOOS: "darwin", Exe: exe, Run: r}
	if err := in.Apply("/tmp/x.dmg"); err != nil {
		t.Fatal(err)
	}
	if got, _ := os.ReadFile(exe); string(got) != "novo" {
		t.Fatalf(".app não trocado: %q", got)
	}
	if _, err := os.Stat(app + ".old"); err == nil {
		t.Fatal(".old ficou")
	}
	joined := strings.Join(r.calls, "\n")
	if !strings.Contains(joined, "hdiutil attach -nobrowse") || !strings.Contains(joined, "hdiutil detach") {
		t.Fatalf("montagem: %v", r.calls)
	}
}

func TestApplyDarwinFallbackOpensDMG(t *testing.T) {
	_, exe := macApp(t)
	r := &fakeRunner{runErr: map[string]error{"ditto": fmt.Errorf("falhou")}}
	in := Installer{GOOS: "darwin", Exe: exe, Run: r}
	if err := in.Apply("/tmp/x.dmg"); err == nil {
		t.Fatal("devia avisar que não instalou")
	}
	if got, _ := os.ReadFile(exe); string(got) != "velho" {
		t.Fatal("o app antigo tinha que ficar")
	}
	if last := r.calls[len(r.calls)-1]; last != "start open /tmp/x.dmg" {
		t.Fatalf("devia abrir o dmg: %v", r.calls)
	}
	// fora de um .app: abre o dmg direto
	r2 := &fakeRunner{}
	if err := (Installer{GOOS: "darwin", Exe: "/usr/local/bin/arkflix-games", Run: r2}).Apply("/tmp/x.dmg"); err == nil || len(r2.calls) != 1 {
		t.Fatalf("sem .app: %v %v", err, r2.calls)
	}
}
