package core

import (
	"context"
	"errors"
	"time"

	"github.com/stanleygomes/arkflix-games/app/internal/remoteplay"
)

// RemotePlayInfo é a aba "Jogar em outro aparelho": o Sunshine desta
// máquina e o endereço pra digitar no Moonlight.
type RemotePlayInfo struct {
	remoteplay.Status
	Busy bool `json:"busy"` // instalação em andamento
}

const remotePlayTask = "remoteplay"

func (s *Service) remotePlayInfo() RemotePlayInfo {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	s.mu.Lock()
	_, busy := s.tasks[remotePlayTask]
	s.mu.Unlock()
	return RemotePlayInfo{Status: s.opt.RemotePlay.Status(ctx), Busy: busy}
}

func (s *Service) emitRemotePlay() {
	s.opt.Emitter.Emit("remoteplay", s.remotePlayInfo())
}

// RemotePlayStatus diz se o Sunshine está instalado e ligado.
func (s *Service) RemotePlayStatus() RemotePlayInfo {
	return s.remotePlayInfo()
}

// InstallRemotePlay instala o Sunshine. Linux com Flatpak: em segundo plano
// (tarefa "remoteplay"; no fim, evento "remoteplay"). Nos outros: abre a
// página de download oficial (o instalador é do usuário).
func (s *Service) InstallRemotePlay() error {
	if err := s.adult(); err != nil {
		return err
	}
	st := s.RemotePlayStatus()
	if !st.Supported {
		return errors.New(st.Reason)
	}
	if !st.AutoInstall {
		return s.opt.Window.OpenFile(st.Download)
	}
	rp := s.opt.RemotePlay
	started := s.runTaskCtx(remotePlayTask, "Instalando o Sunshine", func(parent context.Context, progress func(float64)) (string, error) {
		ctx, cancel := context.WithTimeout(parent, time.Hour)
		defer cancel()
		defer s.emitRemotePlay()
		return rp.Install(ctx, progress)
	})
	if !started {
		return errors.New("o Sunshine já está sendo instalado")
	}
	return nil
}

// StartRemotePlay liga o Sunshine (fica ligado mesmo se o app fechar).
func (s *Service) StartRemotePlay() error {
	if err := s.adult(); err != nil {
		return err
	}
	if err := s.opt.RemotePlay.Start(s.emitRemotePlay); err != nil {
		return err
	}
	s.emitRemotePlay()
	return nil
}

// StopRemotePlay desliga o Sunshine, se foi o app que ligou.
func (s *Service) StopRemotePlay() error {
	if err := s.adult(); err != nil {
		return err
	}
	err := s.opt.RemotePlay.Stop()
	s.emitRemotePlay()
	return err
}

// OpenRemotePlayPage abre a página local do Sunshine na aba de parear (o PIN
// que o Moonlight mostra). O usuário e a senha dela ficam só no Sunshine.
func (s *Service) OpenRemotePlayPage() error {
	if err := s.adult(); err != nil {
		return err
	}
	return s.opt.Window.OpenFile(s.RemotePlayStatus().PairURL)
}
