// Package launcher monta e roda o comando de cada emulador usando os
// lançadores de build/bin (que já apontam config e saves pros lugares certos).
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

// Spec diz onde estão os lançadores e cores.
type Spec struct {
	Bin        string   // pasta com retroarch, ppsspp, pcsx2
	Cores      string   // pasta com <core>_libretro.so
	Fullscreen bool     // abre o jogo em tela cheia
	RAAppend   []string // configs extras do RetroArch (entrada, atalhos)
}

// Resume escolhe de onde o jogo começa. Vazio = do início.
type Resume struct {
	Slot int    // RetroArch: slot do save state (0-999); -1 = automático
	File string // PPSSPP/PCSX2: arquivo de save state
}

// Command devolve o comando e as variáveis extras pra abrir o jogo.
func Command(sys catalog.System, rom string, spec Spec, resume *Resume) ([]string, []string, error) {
	if rom == "" {
		return nil, nil, errors.New("jogo sem arquivo pra abrir")
	}
	var args, env []string
	switch sys.Emulator {
	case catalog.RetroArch:
		core := filepath.Join(spec.Cores, sys.Core+"_libretro.so")
		if _, err := os.Stat(core); err != nil {
			return nil, nil, fmt.Errorf("core %s não compilado (make core-%s)", sys.Core, sys.Core)
		}
		args = []string{filepath.Join(spec.Bin, "retroarch")}
		if spec.Fullscreen {
			args = append(args, "--fullscreen")
		}
		appendCfg := append([]string{}, spec.RAAppend...)
		if resume != nil {
			if resume.Slot >= 0 {
				args = append(args, fmt.Sprintf("--entryslot=%d", resume.Slot))
			} else {
				auto := filepath.Join(os.TempDir(), "arkflix-resume.cfg")
				if err := os.WriteFile(auto, []byte("savestate_auto_load = \"true\"\n"), 0o644); err != nil {
					return nil, nil, err
				}
				appendCfg = append(appendCfg, auto)
			}
		}
		if len(appendCfg) > 0 {
			env = append(env, "ARKFLIX_RA_APPEND="+strings.Join(appendCfg, "|"))
		}
		args = append(args, "-L", core, rom)
	case catalog.PPSSPP:
		args = []string{filepath.Join(spec.Bin, "ppsspp")}
		if spec.Fullscreen {
			args = append(args, "--fullscreen")
		}
		if resume != nil && resume.File != "" {
			args = append(args, "--state="+resume.File)
		}
		args = append(args, rom)
	case catalog.PCSX2:
		args = []string{filepath.Join(spec.Bin, "pcsx2"), "-batch"}
		if spec.Fullscreen {
			args = append(args, "-fullscreen")
		}
		if resume != nil && resume.File != "" {
			args = append(args, "-statefile", resume.File)
		}
		args = append(args, "--", rom)
	default:
		return nil, nil, fmt.Errorf("emulador desconhecido: %s", sys.Emulator)
	}
	if _, err := os.Stat(args[0]); err != nil {
		return nil, nil, fmt.Errorf("%s não encontrado: rode make build", filepath.Base(args[0]))
	}
	return args, env, nil
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
