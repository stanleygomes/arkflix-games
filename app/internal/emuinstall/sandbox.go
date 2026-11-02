package emuinstall

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"time"
)

// Sandbox finge a instalação dentro das pastas de teste (--bin/--cores ou
// --fixture): nunca toca o sistema. Cada passo demora Delay e só cria o que
// falta: um lançador falso (registra a chamada em calls.log, como os do
// fixture) e cores vazios. ARKFLIX_FAKE_INSTALL_FAIL=<id> faz esse passo falhar.
type Sandbox struct {
	Bin, Cores string
	Delay      time.Duration // 0 = 150 ms
}

// fakeEmulator: o mesmo lançador de mentira do fixture.
const fakeEmulator = `#!/bin/sh
dir=$(dirname "$0")
printf '%s|%s\n' "$(basename "$0")" "$*" >> "$dir/calls.log"
sleep "${ARKFLIX_FAKE_SLEEP:-0.3}"
exit "${ARKFLIX_FAKE_EXIT:-0}"
`

// Status: o que existe nas pastas de teste.
func (s *Sandbox) Status(context.Context) Status {
	st := Status{Platform: runtime.GOOS, CanInstall: true, CoresDir: s.Cores, Cores: NeededCores()}
	for _, emu := range emulators {
		p := filepath.Join(s.Bin, emu)
		st.Emulators = append(st.Emulators, Emulator{ID: emu, Name: names[emu], Installed: isFile(p), Path: p})
	}
	for i, c := range st.Cores {
		st.Cores[i].Installed = isFile(s.corePath(c.ID))
	}
	return st
}

func (s *Sandbox) corePath(core string) string {
	return filepath.Join(s.Cores, core+"_libretro.so")
}

// Install simula os passos com progresso e cria os arquivos que faltam.
func (s *Sandbox) Install(ctx context.Context, which []string, progress func(float64)) (string, error) {
	if err := CheckNames(which); err != nil {
		return "", err
	}
	emus, cores := plan(s.Status(ctx), which)
	fail := os.Getenv("ARKFLIX_FAKE_INSTALL_FAIL")
	var steps []step
	for _, emu := range emus {
		steps = append(steps, step{name: names[emu], weight: 10, run: s.fake(emu, fail, filepath.Join(s.Bin, emu), []byte(fakeEmulator), 0o755)})
	}
	for _, c := range cores {
		steps = append(steps, step{name: Cores, weight: 1, run: s.fake(c, fail, s.corePath(c), nil, 0o644)})
	}
	return runSteps(ctx, steps, progress)
}

func (s *Sandbox) fake(id, fail, path string, body []byte, mode os.FileMode) func(context.Context, func(float64)) error {
	return func(ctx context.Context, progress func(float64)) error {
		delay := s.Delay
		if delay == 0 {
			delay = 150 * time.Millisecond
		}
		for i := 1; i <= 3; i++ {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(delay / 3):
			}
			progress(float64(i) / 3)
		}
		if slices.Contains([]string{id, "all"}, fail) {
			return errors.New(id + ": falha simulada")
		}
		if isFile(path) {
			return nil // não pisa no lançador do fixture
		}
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return err
		}
		return os.WriteFile(path, body, mode)
	}
}
