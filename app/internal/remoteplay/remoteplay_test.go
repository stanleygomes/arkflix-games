package remoteplay

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

// sys é um sistema de mentira: comandos gravados, Flatpak instalado ou não.
type sys struct {
	mu        sync.Mutex
	cmds      []string
	flatpak   bool // flatpak --version funciona
	installed bool // flatpak info do Sunshine funciona
	extraFail bool // additional-install.sh falha (senha negada)
	spawned   []string
	proc      *fakeProc
}

func (y *sys) run(_ context.Context, name string, args ...string) ([]byte, error) {
	y.mu.Lock()
	defer y.mu.Unlock()
	line := name + " " + strings.Join(args, " ")
	y.cmds = append(y.cmds, line)
	switch {
	case !y.flatpak:
		return nil, errors.New("flatpak: não encontrado")
	case args[0] == "info" && !y.installed:
		return nil, errors.New("não instalado")
	case args[0] == "install":
		y.installed = true
	case strings.Contains(line, "additional-install.sh") && y.extraFail:
		return nil, errors.New("pkexec: negado")
	case args[0] == "kill":
		if y.proc != nil {
			y.proc.Kill()
		}
	}
	return nil, nil
}

func (y *sys) spawn(name string, args ...string) (Proc, error) {
	y.mu.Lock()
	defer y.mu.Unlock()
	y.spawned = append(y.spawned, name+" "+strings.Join(args, " "))
	y.proc = &fakeProc{done: make(chan struct{})}
	return y.proc, nil
}

func (y *sys) has(cmd string) bool {
	y.mu.Lock()
	defer y.mu.Unlock()
	for _, c := range y.cmds {
		if c == cmd {
			return true
		}
	}
	return false
}

type fakeProc struct {
	once sync.Once
	done chan struct{}
}

func (p *fakeProc) Wait() error { <-p.done; return nil }
func (p *fakeProc) Kill() error { p.once.Do(func() { close(p.done) }); return nil }

func linux(y *sys) *Sunshine {
	return &Sunshine{GOOS: "linux", Home: "/home/x", Run: y.run, Spawn: y.spawn,
		LookExe: func(string) (string, error) { return "", errors.New("não tem") },
		Probe:   func() bool { return false }, LanIP: func() string { return "10.0.0.5" },
		Flathub: "https://flathub.test/repo"}
}

func TestLinuxInstallStartStop(t *testing.T) {
	y := &sys{flatpak: true}
	s := linux(y)
	ctx := context.Background()
	st := s.Status(ctx)
	if !st.Supported || !st.AutoInstall || st.Installed || st.Running || st.LanIP != "10.0.0.5" || st.PairURL != "https://localhost:47990/pin" {
		t.Fatalf("antes: %+v", st)
	}
	if err := s.Start(nil); err == nil {
		t.Fatal("ligar sem instalar")
	}
	var last float64
	msg, err := s.Install(ctx, func(p float64) { last = p })
	if err != nil || msg != "Sunshine instalado" || last != 1 {
		t.Fatalf("instalar: %q %v %v", msg, err, last)
	}
	for _, c := range []string{
		"flatpak remote-add --user --if-not-exists flathub https://flathub.test/repo",
		"flatpak install --user -y --noninteractive flathub " + FlatpakID,
		"flatpak run --command=additional-install.sh " + FlatpakID,
	} {
		if !y.has(c) {
			t.Fatalf("faltou %q em %v", c, y.cmds)
		}
	}
	if err := s.Stop(); err == nil {
		t.Fatal("desligar o que o app não ligou")
	}
	exited := make(chan struct{})
	if err := s.Start(func() { close(exited) }); err != nil {
		t.Fatal(err)
	}
	if len(y.spawned) != 1 || y.spawned[0] != "flatpak run "+FlatpakID {
		t.Fatalf("processo: %v", y.spawned)
	}
	if st := s.Status(ctx); !st.Installed || !st.Running || !st.Started {
		t.Fatalf("ligado: %+v", st)
	}
	if err := s.Start(nil); err == nil {
		t.Fatal("ligar duas vezes")
	}
	if err := s.Stop(); err != nil {
		t.Fatal(err)
	}
	if !y.has("flatpak kill " + FlatpakID) {
		t.Fatalf("sem flatpak kill: %v", y.cmds)
	}
	select {
	case <-exited:
	case <-time.After(2 * time.Second):
		t.Fatal("onExit não rodou")
	}
	if st := s.Status(ctx); st.Running || st.Started {
		t.Fatalf("desligado: %+v", st)
	}
}

// Sem a senha do sistema, o Sunshine fica instalado e o aviso diz o comando.
func TestLinuxInstallExtraStepFails(t *testing.T) {
	y := &sys{flatpak: true, extraFail: true}
	msg, err := linux(y).Install(context.Background(), func(float64) {})
	if err != nil || !strings.Contains(msg, "additional-install.sh") {
		t.Fatalf("%q %v", msg, err)
	}
}

func TestLinuxWithoutFlatpak(t *testing.T) {
	y := &sys{}
	s := linux(y)
	s.LookExe = func(string) (string, error) { return "/usr/bin/sunshine", nil } // pacote da distro
	st := s.Status(context.Background())
	if st.AutoInstall || st.Reason == "" || st.Download == "" || !st.Installed {
		t.Fatalf("%+v", st)
	}
	if _, err := s.Install(context.Background(), func(float64) {}); err == nil {
		t.Fatal("instalar sem Flatpak")
	}
	if err := s.Start(nil); err != nil || y.spawned[0] != "/usr/bin/sunshine " {
		t.Fatalf("ligar o nativo: %v %v", err, y.spawned)
	}
	if err := s.Stop(); err != nil { // sem flatpak kill: mata o processo
		t.Fatal(err)
	}
}

func TestWindowsUsesInstaller(t *testing.T) {
	dir := t.TempDir()
	exe := filepath.Join(dir, "Sunshine", "sunshine.exe")
	y := &sys{}
	s := linux(y)
	s.GOOS = "windows"
	s.Env = func(k string) string {
		if k == "ProgramFiles" {
			return dir
		}
		return ""
	}
	if st := s.Status(context.Background()); st.AutoInstall || st.Installed || st.Download != Download {
		t.Fatalf("antes: %+v", st)
	}
	os.MkdirAll(filepath.Dir(exe), 0o755)
	os.WriteFile(exe, []byte("x"), 0o755)
	s.Probe = func() bool { return true } // o serviço do MSI já liga sozinho
	st := s.Status(context.Background())
	if !st.Installed || !st.Running || st.Started {
		t.Fatalf("depois: %+v", st)
	}
	if err := s.Start(nil); err == nil {
		t.Fatal("já ligado pelo serviço")
	}
}

func TestUnsupported(t *testing.T) {
	s := linux(&sys{})
	s.GOOS = "plan9"
	if st := s.Status(context.Background()); st.Supported || st.Reason == "" {
		t.Fatalf("%+v", st)
	}
}
