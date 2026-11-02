package updater

import (
	"archive/tar"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
)

// InstallCmd é o comando que reinstala na mão (Linux) quando não dá sozinho.
const InstallCmd = "curl -fsSL https://raw.githubusercontent.com/stanleygomes/arkflix-games/master/install.sh | bash"

// Runner roda os comandos do sistema (hdiutil, ditto, o instalador).
type Runner interface {
	Run(name string, args ...string) error   // espera terminar
	Start(name string, args ...string) error // solto: sobrevive ao app
}

// ExecRunner roda de verdade.
type ExecRunner struct{}

// Run espera o comando e devolve a saída junto do erro.
func (ExecRunner) Run(name string, args ...string) error {
	out, err := exec.Command(name, args...).CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s: %w: %s", name, err, strings.TrimSpace(string(out)))
	}
	return nil
}

// Start larga o processo em outro grupo/sessão, sem janela.
func (ExecRunner) Start(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	detach(cmd)
	if err := cmd.Start(); err != nil {
		return err
	}
	return cmd.Process.Release()
}

// Installer põe o arquivo baixado no lugar do app que está rodando.
type Installer struct {
	GOOS string
	Exe  string // binário em uso, sem links
	Run  Runner
}

// Current monta o instalador do binário em uso.
func Current(goos string) (Installer, error) {
	exe, err := os.Executable()
	if err == nil {
		exe, err = filepath.EvalSymlinks(exe)
	}
	if err != nil {
		return Installer{}, fmt.Errorf("não achei o executável do app: %w", err)
	}
	return Installer{GOOS: goos, Exe: exe, Run: ExecRunner{}}, nil
}

// Check diz, antes de baixar, se dá pra instalar sozinho. Só o Linux tem
// esse limite: no macOS sem permissão o .dmg abre pro usuário; no Windows o
// instalador é na pasta do usuário.
func (in Installer) Check() error {
	if in.GOOS != "linux" {
		return nil
	}
	prefix, err := linuxPrefix(in.Exe)
	if err != nil {
		return err
	}
	return writable(filepath.Join(prefix, "bin"))
}

// Apply instala o arquivo baixado (chamado ao fechar o app).
func (in Installer) Apply(file string) error {
	switch in.GOOS {
	case "linux":
		return in.applyLinux(file)
	case "windows":
		name, args := windowsCommand(file)
		return in.Run.Start(name, args...)
	case "darwin":
		return in.applyDarwin(file)
	}
	return fmt.Errorf("atualização automática não existe pra %s", in.GOOS)
}

// ApplyAndRestart instala agora e abre o app de novo depois que este fechar
// (botão "Atualizar e reiniciar"). Quem chama fecha o app logo em seguida.
func (in Installer) ApplyAndRestart(file string, args []string) error {
	switch in.GOOS {
	case "windows":
		// o setup só roda depois que o app fecha; o mesmo script reabre
		name, cmd := windowsCommand(file)
		cmd[len(cmd)-1] += "; Start-Process -FilePath " + psQuote(in.Exe)
		return in.Run.Start(name, cmd...)
	case "linux":
		if err := in.applyLinux(file); err != nil {
			return err
		}
		// espera este fechar pra não ter duas janelas (e o controle disputado)
		return in.Run.Start("sh", append([]string{"-c", `sleep 1; exec "$0" "$@"`, in.Exe}, args...)...)
	case "darwin":
		if err := in.applyDarwin(file); err != nil {
			return err
		}
		app, _ := appBundle(in.Exe)
		return in.Run.Start("sh", "-c", `sleep 1; open -n "$0"`, app)
	}
	return fmt.Errorf("atualização automática não existe pra %s", in.GOOS)
}

func psQuote(s string) string { return "'" + strings.ReplaceAll(s, "'", "''") + "'" }

// linuxPrefix: o app mora em <prefix>/bin/arkflix-games (build/prefix do
// install.sh), o mesmo layout do tarball (prefix/bin/arkflix-games).
func linuxPrefix(exe string) (string, error) {
	if filepath.Base(filepath.Dir(exe)) != "bin" {
		return "", fmt.Errorf("instalação fora do padrão (%s): atualize com %s", exe, InstallCmd)
	}
	return filepath.Dir(filepath.Dir(exe)), nil
}

func writable(dir string) error {
	f, err := os.CreateTemp(dir, ".arkflix-w-*")
	if err != nil {
		return fmt.Errorf("sem permissão pra atualizar %s: atualize com %s", dir, InstallCmd)
	}
	f.Close()
	return os.Remove(f.Name())
}

func (in Installer) applyLinux(file string) error {
	prefix, err := linuxPrefix(in.Exe)
	if err != nil {
		return err
	}
	if err := writable(filepath.Join(prefix, "bin")); err != nil {
		return err
	}
	f, err := os.Open(file)
	if err != nil {
		return err
	}
	defer f.Close()
	return Extract(f, prefix)
}

// Extract abre o tar.gz da release sobre prefix: "prefix/bin/x" vai pra
// <prefix>/bin/x. Tudo vai pra temporários antes e só no fim é renomeado
// (troca atômica por arquivo: o binário em uso segue intacto até o rename).
// Caminho que escapa da pasta, link e afins são recusados.
func Extract(r io.Reader, prefix string) error {
	gz, err := gzip.NewReader(r)
	if err != nil {
		return fmt.Errorf("pacote inválido: %w", err)
	}
	tr := tar.NewReader(gz)
	var staged [][2]string // temporário, destino
	defer func() {
		for _, s := range staged {
			os.Remove(s[0])
		}
	}()
	for {
		h, err := tr.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return fmt.Errorf("pacote inválido: %w", err)
		}
		target, err := entryTarget(prefix, h.Name)
		if err != nil {
			return err
		}
		switch {
		case target == "":
		case h.Typeflag == tar.TypeDir:
			if err := os.MkdirAll(target, 0o755); err != nil {
				return err
			}
		case h.Typeflag == tar.TypeReg:
			tmp, err := stage(tr, target, os.FileMode(h.Mode)&0o755|0o600)
			if err != nil {
				return err
			}
			staged = append(staged, [2]string{tmp, target})
		default:
			return fmt.Errorf("pacote com entrada não suportada: %s", h.Name)
		}
	}
	if len(staged) == 0 {
		return errors.New("pacote vazio")
	}
	for _, s := range staged {
		if err := os.Rename(s[0], s[1]); err != nil {
			return err
		}
	}
	return nil
}

// entryTarget tira o primeiro componente ("prefix/") e prende em prefix.
// "" = a própria raiz (nada a fazer).
func entryTarget(prefix, name string) (string, error) {
	clean := path.Clean(strings.ReplaceAll(name, "\\", "/"))
	if path.IsAbs(clean) || clean == ".." || strings.HasPrefix(clean, "../") {
		return "", fmt.Errorf("pacote com caminho perigoso: %s", name)
	}
	_, rest, _ := strings.Cut(clean, "/")
	if rest == "" {
		return "", nil
	}
	target := filepath.Join(prefix, filepath.FromSlash(rest))
	if !strings.HasPrefix(target, filepath.Clean(prefix)+string(filepath.Separator)) {
		return "", fmt.Errorf("pacote com caminho perigoso: %s", name)
	}
	return target, nil
}

// stage grava o conteúdo num temporário ao lado do destino (mesmo disco: o
// rename é atômico).
func stage(r io.Reader, target string, mode os.FileMode) (string, error) {
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return "", err
	}
	f, err := os.CreateTemp(filepath.Dir(target), "."+filepath.Base(target)+".new-*")
	if err != nil {
		return "", err
	}
	_, err = io.Copy(f, r)
	if cerr := f.Close(); err == nil {
		err = cerr
	}
	if err == nil {
		err = os.Chmod(f.Name(), mode)
	}
	if err != nil {
		os.Remove(f.Name())
		return "", err
	}
	return f.Name(), nil
}

// windowsCommand: o setup NSIS em modo silencioso (/S), 3 s depois, pra o
// app já ter fechado e soltado o .exe.
func windowsCommand(setup string) (string, []string) {
	script := "Start-Sleep -Seconds 3; Start-Process -FilePath " + psQuote(setup) + " -ArgumentList '/S' -Wait"
	return "powershell", []string{"-NoProfile", "-NonInteractive", "-WindowStyle", "Hidden", "-Command", script}
}

// appBundle: .../arkflix-games.app/Contents/MacOS/arkflix-games → o .app.
func appBundle(exe string) (string, error) {
	app := filepath.Dir(filepath.Dir(filepath.Dir(exe)))
	if filepath.Ext(app) != ".app" {
		return "", fmt.Errorf("o app não está num .app (%s)", exe)
	}
	return app, nil
}

// applyDarwin troca o .app pelo do .dmg; sem permissão ou dando errado,
// abre o .dmg pro usuário arrastar.
func (in Installer) applyDarwin(dmg string) error {
	err := in.replaceApp(dmg)
	if err == nil {
		return nil
	}
	if oerr := in.Run.Start("open", dmg); oerr != nil {
		return fmt.Errorf("%w (e o instalador não abriu: %v)", err, oerr)
	}
	return fmt.Errorf("%w: abri o instalador pra você", err)
}

func (in Installer) replaceApp(dmg string) error {
	app, err := appBundle(in.Exe)
	if err != nil {
		return err
	}
	if err := writable(filepath.Dir(app)); err != nil {
		return err
	}
	mnt, err := os.MkdirTemp("", "arkflix-dmg-")
	if err != nil {
		return err
	}
	defer os.Remove(mnt)
	if err := in.Run.Run("hdiutil", "attach", "-nobrowse", "-readonly", "-noautoopen", "-mountpoint", mnt, dmg); err != nil {
		return err
	}
	defer in.Run.Run("hdiutil", "detach", "-quiet", mnt)
	stage, old := app+".new", app+".old"
	os.RemoveAll(stage)
	os.RemoveAll(old)
	if err := in.Run.Run("ditto", filepath.Join(mnt, "arkflix-games.app"), stage); err != nil {
		os.RemoveAll(stage)
		return err
	}
	if err := os.Rename(app, old); err != nil {
		os.RemoveAll(stage)
		return err
	}
	if err := os.Rename(stage, app); err != nil {
		os.Rename(old, app) // volta o antigo
		return err
	}
	return os.RemoveAll(old)
}
