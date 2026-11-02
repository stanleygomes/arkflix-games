package core

import (
	"context"
	"errors"
	"time"

	"github.com/stanleygomes/arkflix-games/app/internal/catalog"
	"github.com/stanleygomes/arkflix-games/app/internal/emuinstall"
)

// EmulatorsInfo é a tela de emuladores: o que está instalado e se dá pra
// instalar/atualizar por aqui.
type EmulatorsInfo struct {
	emuinstall.Status
	Busy bool `json:"busy"` // instalação em andamento
}

// emuInstaller: o instalador de verdade ou o de mentira dos testes.
type emuInstaller interface {
	Status(ctx context.Context) emuinstall.Status
	Install(ctx context.Context, which []string, progress func(float64)) (string, error)
}

// installer: com --bin/--fixture, tudo acontece nas pastas de teste.
func (s *Service) installer() emuInstaller {
	if spec := s.opt.Emulators; spec.Bin != "" {
		return &emuinstall.Sandbox{Bin: spec.Bin, Cores: spec.Cores}
	}
	cfg := s.opt.Store.Settings().Emulators
	in := emuinstall.New()
	in.Exe = map[string]string{catalog.RetroArch: cfg.RetroArch, catalog.PPSSPP: cfg.PPSSPP, catalog.PCSX2: cfg.PCSX2}
	in.CoresDir = cfg.Cores
	return in
}

const emulatorsTask = "emulators"

// EmulatorStatus diz quais emuladores e cores estão instalados.
func (s *Service) EmulatorStatus() EmulatorsInfo {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	s.mu.Lock()
	_, busy := s.tasks[emulatorsTask]
	s.mu.Unlock()
	return EmulatorsInfo{Status: s.installer().Status(ctx), Busy: busy}
}

// InstallEmulators instala em segundo plano (evento "task" com id
// "emulators"). which vazio = o que falta; com nomes (retroarch, ppsspp,
// pcsx2, cores) reinstala/atualiza esses. No fim sai o evento "emulators".
func (s *Service) InstallEmulators(which []string) error {
	if err := s.adult(); err != nil {
		return err
	}
	if err := emuinstall.CheckNames(which); err != nil {
		return err
	}
	in := s.installer()
	started := s.runTaskCtx(emulatorsTask, "Instalando emuladores", func(parent context.Context, progress func(float64)) (string, error) {
		ctx, cancel := context.WithTimeout(parent, time.Hour)
		defer cancel()
		// Busy fica falso: a tarefa está acabando (contexto próprio: o da
		// instalação pode ter sido cancelado)
		defer func() {
			sctx, done := context.WithTimeout(context.Background(), 30*time.Second)
			defer done()
			s.opt.Emitter.Emit("emulators", EmulatorsInfo{Status: in.Status(sctx)})
		}()
		return in.Install(ctx, which, progress)
	})
	if !started {
		return errors.New("os emuladores já estão sendo instalados")
	}
	return nil
}
