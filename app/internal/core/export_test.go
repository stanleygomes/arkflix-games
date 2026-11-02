package core

import (
	"time"

	"github.com/stanleygomes/arkflix-games/app/internal/updater"
)

// Ganchos só dos testes: sistema fixo e instalador falso, pra nunca mexer
// no binário que está rodando.

func SetUpdatePlatform(s *Service, goos, goarch string) {
	s.upd.mu.Lock()
	defer s.upd.mu.Unlock()
	s.upd.goos, s.upd.goarch = goos, goarch
}

func SetUpdateInstaller(s *Service, fn func() (updater.Installer, error)) {
	s.upd.mu.Lock()
	defer s.upd.mu.Unlock()
	s.upd.installer = fn
}

func SetCheckDelay(d time.Duration) { checkDelay = d }
