// Package remoteplay ajuda a jogar em outro aparelho da casa: o Sunshine
// (LizardByte) transmite esta máquina e o Moonlight, na TV ou no celular,
// recebe. O app não transmite nada: só instala, liga, desliga e mostra como
// parear.
//
// Linux: o Flatpak do Flathub (dev.lizardbyte.app.Sunshine), instalado só pro
// usuário, ou o pacote da distribuição se já estiver no PATH. Windows e macOS:
// o instalador oficial (MSI/Homebrew), que o usuário roda pela página de
// download. O pareamento é pela página web local do Sunshine (aba PIN): o app
// nunca pede nem guarda a senha dela.
//
// Tudo que toca o sistema (comandos, processos, rede) fica no Sunshine, pra
// os testes trocarem por falsos.
package remoteplay

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	FlatpakID = "dev.lizardbyte.app.Sunshine"
	Port      = 47990 // página web local do Sunshine (HTTPS, certificado próprio)
	// Download: instaladores oficiais (MSI no Windows, DMG no macOS).
	Download = "https://github.com/LizardByte/Sunshine/releases/latest"
	// Docs: passo a passo oficial (Homebrew no macOS, primeiros passos).
	Docs = "https://docs.lizardbyte.dev/projects/sunshine/latest/md_docs_2getting__started.html"
)

// URL: a página web local (usuário e senha são criados no primeiro acesso).
var URL = "https://localhost:" + strconv.Itoa(Port)

// Status é o que a aba "Jogar em outro aparelho" mostra.
type Status struct {
	Supported   bool   `json:"supported"`
	Platform    string `json:"platform"`    // linux, windows, darwin
	AutoInstall bool   `json:"autoInstall"` // instala pelo app (Flatpak); senão, página de download
	Installed   bool   `json:"installed"`
	Running     bool   `json:"running"`
	Started     bool   `json:"started"` // foi o app que ligou (dá pra desligar daqui)
	URL         string `json:"url"`     // página web local
	PairURL     string `json:"pairUrl"` // aba de parear com o PIN do Moonlight
	Download    string `json:"download,omitempty"`
	LanIP       string `json:"lanIP"` // o endereço pra digitar no Moonlight
	Reason      string `json:"reason,omitempty"`
}

// Backend: o Sunshine de verdade ou o de mentira dos testes.
type Backend interface {
	Status(ctx context.Context) Status
	Install(ctx context.Context, progress func(float64)) (string, error)
	// Start liga em segundo plano; onExit roda quando o processo termina.
	Start(onExit func()) error
	Stop() error
}

// Runner roda um comando e devolve a saída (stdout+stderr).
type Runner func(ctx context.Context, name string, args ...string) ([]byte, error)

// Proc é um processo filho: esperar e matar.
type Proc interface {
	Wait() error
	Kill() error
}

// Spawner inicia um processo sem esperar (argumentos separados, sem shell).
type Spawner func(name string, args ...string) (Proc, error)

// Sunshine controla o Sunshine desta máquina. Campos vazios = padrão (New).
type Sunshine struct {
	GOOS    string
	Home    string
	Env     func(string) string
	Run     Runner
	Spawn   Spawner
	LookExe func(string) (string, error)
	Probe   func() bool   // a página web está respondendo
	LanIP   func() string // IPv4 desta máquina na rede da casa
	Flathub string        // .flatpakrepo do Flathub

	mu   sync.Mutex
	proc Proc // o Sunshine que o app ligou (nil: nenhum)
}

// New monta o Sunshine deste sistema.
func New() *Sunshine {
	home, _ := os.UserHomeDir()
	return &Sunshine{
		GOOS: runtime.GOOS, Home: home, Env: os.Getenv, Run: execRun, Spawn: execSpawn,
		LookExe: exec.LookPath, Probe: probe, LanIP: lanIP,
		Flathub: "https://dl.flathub.org/repo/flathub.flatpakrepo",
	}
}

func execRun(ctx context.Context, name string, args ...string) ([]byte, error) {
	out, err := exec.CommandContext(ctx, name, args...).CombinedOutput()
	if err != nil {
		msg := strings.TrimSpace(string(out))
		if len(msg) > 300 {
			msg = "…" + msg[len(msg)-300:]
		}
		return out, fmt.Errorf("%s: %w %s", name, err, msg)
	}
	return out, nil
}

type execProc struct{ cmd *exec.Cmd }

func (p execProc) Wait() error { return p.cmd.Wait() }
func (p execProc) Kill() error { return p.cmd.Process.Kill() }

// execSpawn: o processo continua se o app fechar (o usuário pode querer manter).
func execSpawn(name string, args ...string) (Proc, error) {
	cmd := exec.Command(name, args...)
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	return execProc{cmd}, nil
}

// probe: alguém escuta na porta da página web local.
func probe() bool {
	c, err := net.DialTimeout("tcp", net.JoinHostPort("127.0.0.1", strconv.Itoa(Port)), 400*time.Millisecond)
	if err != nil {
		return false
	}
	c.Close()
	return true
}

// lanIP: o IPv4 da rota padrão (UDP não manda nada); sem rota, o primeiro
// endereço privado das interfaces.
func lanIP() string {
	if c, err := net.Dial("udp4", "192.0.2.1:9"); err == nil {
		defer c.Close()
		if a, ok := c.LocalAddr().(*net.UDPAddr); ok && !a.IP.IsLoopback() && !a.IP.IsUnspecified() {
			return a.IP.String()
		}
	}
	addrs, _ := net.InterfaceAddrs()
	for _, a := range addrs {
		if n, ok := a.(*net.IPNet); ok && n.IP.To4() != nil && n.IP.IsPrivate() {
			return n.IP.String()
		}
	}
	return ""
}

func (s *Sunshine) env(k string) string {
	if s.Env == nil {
		return ""
	}
	return s.Env(k)
}

// kind: como o Sunshine está instalado aqui.
type kind int

const (
	none    kind = iota
	flatpak      // Flatpak (usuário ou sistema)
	native       // executável (pacote da distro, MSI, Homebrew, DMG)
)

func (s *Sunshine) hasFlatpak(ctx context.Context) bool {
	_, err := s.Run(ctx, "flatpak", "--version")
	return err == nil
}

// find diz como está instalado e, se nativo, o executável.
func (s *Sunshine) find(ctx context.Context) (kind, string) {
	if s.GOOS == "linux" {
		if _, err := s.Run(ctx, "flatpak", "info", FlatpakID); err == nil {
			return flatpak, ""
		}
	}
	for _, p := range s.candidates() {
		if isFile(p) {
			return native, p
		}
	}
	if s.LookExe != nil {
		if p, err := s.LookExe("sunshine"); err == nil {
			return native, p
		}
	}
	return none, ""
}

// candidates: onde os instaladores oficiais deixam o executável.
func (s *Sunshine) candidates() []string {
	switch s.GOOS {
	case "windows":
		var out []string
		for _, k := range []string{"ProgramFiles", "ProgramW6432"} {
			if d := s.env(k); d != "" {
				out = append(out, filepath.Join(d, "Sunshine", "sunshine.exe"))
			}
		}
		return out
	case "darwin":
		return []string{
			"/opt/homebrew/bin/sunshine", "/usr/local/bin/sunshine",
			"/Applications/Sunshine.app/Contents/MacOS/sunshine",
			filepath.Join(s.Home, "Applications", "Sunshine.app", "Contents", "MacOS", "sunshine"),
		}
	}
	return nil
}

func isFile(p string) bool {
	st, err := os.Stat(p)
	return err == nil && !st.IsDir()
}

// Status: instalado, ligado e o endereço pra digitar no Moonlight.
func (s *Sunshine) Status(ctx context.Context) Status {
	st := Status{Platform: s.GOOS, URL: URL, PairURL: URL + "/pin"}
	switch s.GOOS {
	case "linux":
		st.Supported = true
		st.AutoInstall = s.hasFlatpak(ctx)
		if !st.AutoInstall {
			st.Reason = "falta o Flatpak: instale o Sunshine pelo gerenciador de pacotes"
			st.Download = Docs
		}
	case "windows":
		st.Supported, st.Download = true, Download
	case "darwin":
		st.Supported, st.Download = true, Docs
	default:
		st.Reason = "o Sunshine não roda neste sistema"
		return st
	}
	k, _ := s.find(ctx)
	st.Installed = k != none
	s.mu.Lock()
	st.Started = s.proc != nil
	s.mu.Unlock()
	st.Running = st.Started || (s.Probe != nil && s.Probe())
	if s.LanIP != nil {
		st.LanIP = s.LanIP()
	}
	return st
}

// Install instala o Flatpak só pro usuário (sem sudo) e roda a etapa extra
// do Sunshine (regras de udev pro controle virtual, pede a senha do sistema).
func (s *Sunshine) Install(ctx context.Context, progress func(float64)) (string, error) {
	if s.GOOS != "linux" || !s.hasFlatpak(ctx) {
		return "", errors.New("instalação automática só no Linux com Flatpak: use a página de download")
	}
	step := func(d time.Duration, args ...string) error {
		c, cancel := context.WithTimeout(ctx, d)
		defer cancel()
		_, err := s.Run(c, "flatpak", args...)
		return err
	}
	progress(0.02)
	if err := step(2*time.Minute, "remote-add", "--user", "--if-not-exists", "flathub", s.Flathub); err != nil {
		return "", fmt.Errorf("não deu pra adicionar o Flathub: %w", err)
	}
	progress(0.1)
	if err := step(30*time.Minute, "install", "--user", "-y", "--noninteractive", "flathub", FlatpakID); err != nil {
		return "", fmt.Errorf("Sunshine: %w", err)
	}
	progress(0.85)
	// regras do controle virtual: o script do próprio Sunshine pede a senha
	// do sistema (pkexec). Sem elas o vídeo funciona, só o controle não.
	if err := step(5*time.Minute, "run", "--command=additional-install.sh", FlatpakID); err != nil {
		if ctx.Err() != nil {
			return "", ctx.Err()
		}
		return "Sunshine instalado. Pro controle funcionar do outro lado, rode no terminal: flatpak run --command=additional-install.sh " + FlatpakID, nil
	}
	progress(1)
	return "Sunshine instalado", nil
}

// Start liga o Sunshine como processo filho (continua se o app fechar).
func (s *Sunshine) Start(onExit func()) error {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.proc != nil || (s.Probe != nil && s.Probe()) {
		return errors.New("o Sunshine já está ligado")
	}
	var p Proc
	var err error
	switch k, exe := s.find(ctx); k {
	case flatpak:
		p, err = s.Spawn("flatpak", "run", FlatpakID)
	case native:
		p, err = s.Spawn(exe)
	default:
		return errors.New("instale o Sunshine primeiro")
	}
	if err != nil {
		return fmt.Errorf("não deu pra ligar o Sunshine: %w", err)
	}
	s.proc = p
	go func() {
		p.Wait()
		s.mu.Lock()
		if s.proc == p {
			s.proc = nil
		}
		s.mu.Unlock()
		if onExit != nil {
			onExit()
		}
	}()
	return nil
}

// Stop desliga só o Sunshine que o app ligou.
func (s *Sunshine) Stop() error {
	s.mu.Lock()
	p := s.proc
	s.mu.Unlock()
	if p == nil {
		return errors.New("o Sunshine não foi ligado pelo app: desligue por onde ligou")
	}
	if s.GOOS == "linux" {
		// o flatpak run deixa o Sunshine num sandbox: o kill do flatpak
		// derruba tudo de uma vez (sem Flatpak, não acha e segue pro Kill)
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		if _, err := s.Run(ctx, "flatpak", "kill", FlatpakID); err == nil {
			return nil
		}
	}
	if err := p.Kill(); err != nil && !errors.Is(err, os.ErrProcessDone) {
		return fmt.Errorf("não deu pra desligar o Sunshine: %w", err)
	}
	return nil
}
