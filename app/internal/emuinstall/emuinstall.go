// Package emuinstall instala e atualiza os emuladores e os cores do RetroArch
// pelo próprio app. Linux: Flatpaks do Flathub + cores do buildbot do libretro
// (o mesmo do make emulators). Windows: os pacotes portáteis oficiais em
// %LOCALAPPDATA%\Programs. macOS: os .app oficiais em ~/Applications.
//
// Tudo que toca o mundo (HTTP, comandos, sistema, pasta pessoal) fica no
// Installer, pra os testes trocarem por servidores e comandos falsos.
package emuinstall

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strings"

	"github.com/stanleygomes/arkflix-games/app/internal/catalog"
	"github.com/stanleygomes/arkflix-games/app/internal/launcher"
)

// Versões fixas (troque aqui pra atualizar). O RetroArch usa a estável mais
// nova listada no buildbot e só cai nesta se a listagem falhar; o PCSX2 vem
// sempre do "latest" do GitHub.
const (
	RetroArchStable = "1.22.2" // buildbot.libretro.com/stable/
	PPSSPPVersion   = "1.20.4" // ppsspp.org/download
)

// Cores: o nome de pasta pros pedidos de instalação ("cores" = todos).
const Cores = "cores"

// Status é o que a tela de emuladores mostra.
type Status struct {
	Platform   string     `json:"platform"` // linux, windows, darwin
	CanInstall bool       `json:"canInstall"`
	Reason     string     `json:"reason,omitempty"` // por que não dá pra instalar aqui
	Emulators  []Emulator `json:"emulators"`
	Cores      []Core     `json:"cores"`
	CoresDir   string     `json:"coresDir"`
}

// Emulator: um dos três emuladores.
type Emulator struct {
	ID        string `json:"id"` // retroarch, ppsspp, pcsx2
	Name      string `json:"name"`
	Installed bool   `json:"installed"`
	Path      string `json:"path"`    // executável, ou o id do Flatpak no Linux
	Version   string `json:"version"` // vazio quando não dá pra saber barato
}

// Core: um core do RetroArch que o catálogo usa.
type Core struct {
	ID        string   `json:"id"`      // snes9x
	Systems   []string `json:"systems"` // consoles que dependem dele
	Installed bool     `json:"installed"`
}

// Runner roda um comando e devolve a saída (stdout+stderr).
type Runner func(ctx context.Context, name string, args ...string) ([]byte, error)

// Installer instala de verdade. Campos vazios = padrão do sistema (New).
type Installer struct {
	GOOS, GOARCH string
	Home         string
	Env          func(string) string
	HTTP         *http.Client
	Run          Runner
	Buildbot     string // https://buildbot.libretro.com
	PPSSPP       string // https://www.ppsspp.org/files
	GitHub       string // https://api.github.com
	Flathub      string // .flatpakrepo do Flathub
	// Exe e CoresDir: caminhos escolhidos nas configurações (vazio = padrão).
	Exe      map[string]string
	CoresDir string
}

// New monta o Installer deste sistema.
func New() *Installer {
	home, _ := os.UserHomeDir()
	return &Installer{
		GOOS: runtime.GOOS, GOARCH: runtime.GOARCH, Home: home, Env: os.Getenv,
		HTTP: &http.Client{}, Run: execRun,
		Buildbot: "https://buildbot.libretro.com", PPSSPP: "https://www.ppsspp.org/files",
		GitHub: "https://api.github.com", Flathub: "https://dl.flathub.org/repo/flathub.flatpakrepo",
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

// emulators na ordem de instalação (o RetroArch antes: os cores dependem dele).
var emulators = []string{catalog.RetroArch, catalog.PPSSPP, catalog.PCSX2}

var names = map[string]string{catalog.RetroArch: "RetroArch", catalog.PPSSPP: "PPSSPP", catalog.PCSX2: "PCSX2"}

// NeededCores: os cores que o catálogo usa, sem repetir, com os consoles.
func NeededCores() []Core {
	var out []Core
	for _, sys := range catalog.Systems {
		if sys.Core == "" {
			continue
		}
		i := slices.IndexFunc(out, func(c Core) bool { return c.ID == sys.Core })
		if i < 0 {
			out = append(out, Core{ID: sys.Core})
			i = len(out) - 1
		}
		out[i].Systems = append(out[i].Systems, sys.Slug)
	}
	return out
}

// CheckNames confere os nomes pedidos (retroarch, ppsspp, pcsx2, cores).
func CheckNames(which []string) error {
	for _, w := range which {
		if w != Cores && names[w] == "" {
			return fmt.Errorf("emulador desconhecido: %s", w)
		}
	}
	return nil
}

// support diz se este sistema tem instalação automática (e por que não).
func (in *Installer) support(ctx context.Context) string {
	switch in.GOOS {
	case "linux":
		if in.GOARCH != "amd64" {
			return "instalação automática só no Linux x86_64"
		}
		if _, err := in.Run(ctx, "flatpak", "--version"); err != nil {
			return "falta o Flatpak: instale pelo gerenciador de pacotes (make deps)"
		}
	case "windows":
		if in.GOARCH != "amd64" {
			return "instalação automática só no Windows x64"
		}
	case "darwin":
	default:
		return "instalação automática não disponível neste sistema"
	}
	return ""
}

// Status: o que está instalado agora.
func (in *Installer) Status(ctx context.Context) Status {
	st := Status{Platform: in.GOOS, Reason: in.support(ctx)}
	st.CanInstall = st.Reason == ""
	if in.GOOS == "linux" {
		st.Emulators = in.flatpakStatus(ctx)
	} else {
		for _, emu := range emulators {
			st.Emulators = append(st.Emulators, in.nativeStatus(emu))
		}
	}
	st.CoresDir = in.coresDir()
	st.Cores = NeededCores()
	for i, c := range st.Cores {
		st.Cores[i].Installed = isFile(in.corePath(st.CoresDir, c.ID))
	}
	return st
}

// coresDir: pasta de cores do RetroArch (configurações > ao lado do RetroArch
// no Windows > padrão do sistema).
func (in *Installer) coresDir() string {
	if in.CoresDir != "" {
		return in.CoresDir
	}
	switch in.GOOS {
	case "windows":
		ra := in.find(catalog.RetroArch)
		if ra == "" {
			ra = filepath.Join(in.winTarget(catalog.RetroArch), "retroarch.exe")
		}
		return launcher.DefaultCores("windows", ra)
	case "darwin":
		return filepath.Join(in.Home, "Library", "Application Support", "RetroArch", "cores")
	}
	return filepath.Join(in.Home, ".var", "app", launcher.Flatpaks[catalog.RetroArch], "config", "retroarch", "cores")
}

func (in *Installer) corePath(dir, core string) string {
	return filepath.Join(dir, core+"_libretro"+launcher.CoreExt(in.GOOS))
}

// find: o executável em uso (configurações ou lugar padrão), vazio se não tem.
func (in *Installer) find(emu string) string {
	if exe := in.Exe[emu]; exe != "" && isFile(exe) {
		return exe
	}
	for _, p := range launcher.Candidates(in.GOOS, emu, in.Home, in.env) {
		if isFile(p) {
			return p
		}
	}
	return ""
}

func (in *Installer) env(k string) string {
	if in.Env == nil {
		return ""
	}
	return in.Env(k)
}

func (in *Installer) nativeStatus(emu string) Emulator {
	e := Emulator{ID: emu, Name: names[emu], Path: in.find(emu)}
	e.Installed = e.Path != ""
	if e.Installed && in.GOOS == "darwin" {
		e.Version = plistVersion(appDir(e.Path))
	}
	return e
}

func isFile(p string) bool {
	st, err := os.Stat(p)
	return err == nil && !st.IsDir()
}

// step é um passo da instalação, com peso pra barra de progresso.
type step struct {
	name   string // pra mensagem final
	weight float64
	run    func(ctx context.Context, progress func(float64)) error
}

// Install instala o que falta (which vazio) ou reinstala/atualiza o pedido.
func (in *Installer) Install(ctx context.Context, which []string, progress func(float64)) (string, error) {
	if err := CheckNames(which); err != nil {
		return "", err
	}
	if reason := in.support(ctx); reason != "" {
		return "", errors.New(reason)
	}
	st := in.Status(ctx)
	emus, cores := plan(st, which)
	var steps []step
	if in.GOOS == "linux" && len(emus) > 0 {
		steps = append(steps, step{weight: 0.5, run: in.flathubRemote})
	}
	for _, emu := range emus {
		steps = append(steps, step{name: names[emu], weight: 10, run: in.emulatorStep(emu, st)})
	}
	for _, c := range cores {
		steps = append(steps, step{name: Cores, weight: 1, run: in.coreStep(c)})
	}
	return runSteps(ctx, steps, progress)
}

// plan escolhe emuladores e cores. Vazio = só o que falta.
func plan(st Status, which []string) (emus, cores []string) {
	all := len(which) == 0
	for _, e := range st.Emulators {
		if (all && !e.Installed) || slices.Contains(which, e.ID) {
			emus = append(emus, e.ID)
		}
	}
	for _, c := range st.Cores {
		if (all && !c.Installed) || slices.Contains(which, Cores) {
			cores = append(cores, c.ID)
		}
	}
	return emus, cores
}

func (in *Installer) emulatorStep(emu string, st Status) func(context.Context, func(float64)) error {
	installed := slices.ContainsFunc(st.Emulators, func(e Emulator) bool { return e.ID == emu && e.Installed })
	return func(ctx context.Context, p func(float64)) error {
		switch in.GOOS {
		case "linux":
			return in.flatpakInstall(ctx, emu, installed)
		case "windows":
			return in.winInstall(ctx, emu, p)
		}
		return in.macInstall(ctx, emu, p)
	}
}

func (in *Installer) coreStep(core string) func(context.Context, func(float64)) error {
	return func(ctx context.Context, p func(float64)) error {
		file := core + "_libretro" + launcher.CoreExt(in.GOOS)
		url := fmt.Sprintf("%s/nightly/%s/latest/%s.zip", in.Buildbot, in.coreArch(), file)
		dir := in.coresDir()
		if err := in.fetchCore(ctx, url, file, filepath.Join(dir, file), p); err != nil {
			return fmt.Errorf("core %s: %w", core, err)
		}
		return nil
	}
}

// coreArch: pasta do buildbot com os cores deste sistema.
func (in *Installer) coreArch() string {
	switch in.GOOS {
	case "windows":
		return "windows/x86_64"
	case "darwin":
		if in.GOARCH == "arm64" {
			return "apple/osx/arm64"
		}
		return "apple/osx/x86_64"
	}
	return "linux/x86_64"
}

// runSteps roda tudo (um erro não para os outros) e monta a mensagem final.
func runSteps(ctx context.Context, steps []step, progress func(float64)) (string, error) {
	var total, done float64
	for _, s := range steps {
		total += s.weight
	}
	var ok, failed []string
	cores := 0
	for _, s := range steps {
		w := s.weight
		err := s.run(ctx, func(f float64) { progress((done + min(max(f, 0), 1)*w) / total) })
		done += w
		progress(done / total)
		switch {
		case err != nil && s.name == "":
			return "", err // sem o repositório do Flathub nada mais instala
		case err != nil:
			failed = append(failed, err.Error())
		case s.name == Cores:
			cores++
		case s.name != "":
			ok = append(ok, s.name)
		}
	}
	return summary(ok, cores, failed)
}

func summary(ok []string, cores int, failed []string) (string, error) {
	if cores > 0 {
		ok = append(ok, fmt.Sprintf("%d core%s", cores, plural(cores)))
	}
	msg := ""
	if len(ok) > 0 {
		msg = join(ok) + " pronto"
		if len(ok) > 1 || cores > 1 {
			msg += "s"
		}
	}
	if len(failed) == 0 {
		if msg == "" {
			return "emuladores e cores já instalados", nil
		}
		return msg, nil
	}
	if msg != "" {
		msg += "; "
	}
	return "", errors.New(msg + "falhou: " + strings.Join(failed, "; "))
}

func plural(n int) string {
	if n > 1 {
		return "s"
	}
	return ""
}

// join: "a", "a e b", "a, b e c".
func join(items []string) string {
	if len(items) == 1 {
		return items[0]
	}
	return strings.Join(items[:len(items)-1], ", ") + " e " + items[len(items)-1]
}
