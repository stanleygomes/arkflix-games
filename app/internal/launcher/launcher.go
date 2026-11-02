// Package launcher monta e roda o comando de cada emulador. No Linux usa os
// Flatpaks (org.libretro.RetroArch, org.ppsspp.PPSSPP, net.pcsx2.PCSX2); no
// Windows e no macOS, os emuladores dos instaladores oficiais, nos lugares
// padrão (ou o caminho escolhido nas configurações). Spec.Bin aponta uma pasta
// com executáveis próprios no lugar (os testes usam emuladores falsos assim).
package launcher

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/stanleygomes/arkflix-games/app/internal/catalog"
)

// Flatpaks: o id de cada emulador no Flathub.
var Flatpaks = map[string]string{
	catalog.RetroArch: "org.libretro.RetroArch",
	catalog.PPSSPP:    "org.ppsspp.PPSSPP",
	catalog.PCSX2:     "net.pcsx2.PCSX2",
}

var executables = map[string]string{
	catalog.RetroArch: "retroarch",
	catalog.PPSSPP:    "ppsspp",
	catalog.PCSX2:     "pcsx2",
}

// FlatpakCores é a pasta de cores do RetroArch do Flatpak (o "Online Updater"
// dele baixa pra cá; o make emulators também).
func FlatpakCores() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".var", "app", Flatpaks[catalog.RetroArch], "config", "retroarch", "cores")
}

// Spec diz como abrir os emuladores.
type Spec struct {
	Bin        string            // pasta com retroarch, ppsspp e pcsx2 (testes); vazio = padrão do sistema
	Exe        map[string]string // executável escolhido por emulador (configurações)
	Cores      string            // vazio = cores do emulador no sistema
	GOOS       string            // vazio = este sistema (os testes trocam)
	Fullscreen bool              // abre o jogo em tela cheia
	RAAppend   []string          // configs extras do RetroArch (entrada, saves do perfil)
	Work       string            // pasta pra configs geradas na hora (o Flatpak enxerga a home, não o /tmp)
}

// Resume escolhe de onde o jogo começa. Vazio = do início.
type Resume struct {
	Slot int    // RetroArch: slot do save state (0-999); -1 = automático
	File string // PPSSPP/PCSX2: arquivo de save state
}

// installed: o Flatpak existe? (variável pra os testes trocarem)
var installed = func(id string) bool {
	return exec.Command("flatpak", "info", "--show-ref", id).Run() == nil
}

// program devolve o começo do comando do emulador e confere se ele existe.
func program(emu string, spec Spec) ([]string, error) {
	if spec.Bin != "" {
		path := filepath.Join(spec.Bin, executables[emu])
		if _, err := os.Stat(path); err != nil {
			return nil, fmt.Errorf("%s não encontrado em %s", executables[emu], spec.Bin)
		}
		return []string{path}, nil
	}
	if exe := spec.Exe[emu]; exe != "" {
		if _, err := os.Stat(exe); err != nil {
			return nil, fmt.Errorf("%s não encontrado em %s (configurações › biblioteca)", names[emu], exe)
		}
		return []string{exe}, nil
	}
	if goos := spec.goos(); goos != "linux" {
		if exe := Find(goos, emu); exe != "" {
			return []string{exe}, nil
		}
		return nil, fmt.Errorf("%s não encontrado: instale de %s ou informe o caminho em configurações › biblioteca", names[emu], sites[emu])
	}
	id := Flatpaks[emu]
	if _, err := exec.LookPath("flatpak"); err != nil {
		return nil, errors.New("falta o Flatpak: rode make emulators")
	}
	if !installed(id) {
		return nil, fmt.Errorf("%s não instalado: rode make emulators", id)
	}
	return []string{"flatpak", "run", id}, nil
}

// Command devolve o comando pra abrir o jogo.
func Command(sys catalog.System, rom string, spec Spec, resume *Resume) ([]string, error) {
	if rom == "" {
		return nil, errors.New("jogo sem arquivo pra abrir")
	}
	if _, ok := Flatpaks[sys.Emulator]; !ok {
		return nil, fmt.Errorf("emulador desconhecido: %s", sys.Emulator)
	}
	args, err := program(sys.Emulator, spec)
	if err != nil {
		return nil, err
	}
	switch sys.Emulator {
	case catalog.RetroArch:
		cores := spec.Cores
		if cores == "" {
			cores = DefaultCores(spec.goos(), args[0])
		}
		core := filepath.Join(cores, sys.Core+"_libretro"+CoreExt(spec.goos()))
		if _, err := os.Stat(core); err != nil {
			hint := "rode make emulators"
			if spec.goos() != "linux" {
				hint = "baixe pelo menu do RetroArch (Online Updater › Core Downloader)"
			}
			return nil, fmt.Errorf("core %s não instalado: %s", sys.Core, hint)
		}
		if spec.Fullscreen {
			args = append(args, "--fullscreen")
		}
		appendCfg := append([]string{}, spec.RAAppend...)
		if resume != nil {
			if resume.Slot >= 0 {
				args = append(args, fmt.Sprintf("--entryslot=%d", resume.Slot))
			} else {
				work := spec.Work
				if work == "" {
					work = os.TempDir()
				}
				auto := filepath.Join(work, "arkflix-resume.cfg")
				if err := os.MkdirAll(work, 0o700); err != nil {
					return nil, err
				}
				if err := os.WriteFile(auto, []byte("savestate_auto_load = \"true\"\n"), 0o644); err != nil {
					return nil, err
				}
				appendCfg = append(appendCfg, auto)
			}
		}
		if len(appendCfg) > 0 {
			args = append(args, "--appendconfig="+strings.Join(appendCfg, "|"))
		}
		args = append(args, "-L", core, rom)
	case catalog.PPSSPP:
		if spec.Fullscreen {
			args = append(args, "--fullscreen")
		}
		if resume != nil && resume.File != "" {
			args = append(args, "--state="+resume.File)
		}
		args = append(args, rom)
	case catalog.PCSX2:
		args = append(args, "-batch")
		if spec.Fullscreen {
			args = append(args, "-fullscreen")
		}
		if resume != nil && resume.File != "" {
			args = append(args, "-statefile", resume.File)
		}
		args = append(args, "--", rom)
	}
	return args, nil
}

// Result é como o jogo terminou.
type Result struct {
	Code     int           `json:"code"`
	Duration time.Duration `json:"duration"`
	Tail     string        `json:"tail,omitempty"` // fim da saída, se deu erro
}

type tail struct{ buf []byte }

func (t *tail) Write(p []byte) (int, error) {
	t.buf = append(t.buf, p...)
	if len(t.buf) > 8192 {
		t.buf = t.buf[len(t.buf)-8192:]
	}
	return len(p), nil
}

// Run abre o jogo e espera fechar.
func Run(ctx context.Context, args, env []string) (Result, error) {
	start := time.Now()
	cmd := exec.CommandContext(ctx, args[0], args[1:]...)
	cmd.Env = append(os.Environ(), env...)
	// fechar pelo guia: pede pra sair (SIGTERM) e, se não sair, mata. O
	// WaitDelay também solta quando um filho segura a saída aberta.
	cmd.Cancel = func() error { return terminate(cmd.Process) }
	cmd.WaitDelay = 3 * time.Second
	var out tail
	cmd.Stdout, cmd.Stderr = &out, &out
	err := cmd.Run()
	res := Result{Duration: time.Since(start)}
	var exit *exec.ExitError
	switch {
	case errors.As(err, &exit):
		res.Code = exit.ExitCode()
	case err != nil:
		return res, err
	}
	if res.Code != 0 {
		lines := bytes.Split(bytes.TrimSpace(out.buf), []byte("\n"))
		res.Tail = string(bytes.Join(lines[max(0, len(lines)-12):], []byte("\n")))
	}
	return res, nil
}
