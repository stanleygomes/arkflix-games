package core

import (
	"os"
	"syscall"
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

// SetQuitGrace encurta a espera do "fechar jogo" nos testes.
func SetQuitGrace(d time.Duration) { quitGrace = d }

// Conflict expõe a regra de conflito de saves.
var Conflict = conflict

// SetRestartDelay encurta a espera antes de reiniciar.
func SetRestartDelay(d time.Duration) { restartDelay = d }

// EncodeToken expõe o token de caminho da mídia.
var EncodeToken = encodeToken

// SetMoveOtherDisk faz o mover tratar origem e destino como discos
// diferentes: o rename falha com EXDEV e a cópia entra.
func SetMoveOtherDisk(on bool) {
	if !on {
		moveRename, moveSameDisk = os.Rename, sameDisk
		return
	}
	moveRename = func(a, b string) error { return &os.LinkError{Op: "rename", Old: a, New: b, Err: syscall.EXDEV} }
	moveSameDisk = func(string, string) bool { return false }
}

// SetMoveFree troca a medida de espaço livre do destino (nil = a de verdade).
func SetMoveFree(fn func(string) int64) {
	if fn == nil {
		fn = freeSpace
	}
	moveFree = fn
}

// SetMoveChunk roda fn a cada pedaço copiado (cancelar no meio da cópia).
func SetMoveChunk(fn func(dst string)) {
	if fn == nil {
		fn = func(string) {}
	}
	moveChunk = fn
}

// SetClock troca o relógio do serviço (limite do perfil infantil).
func SetClock(s *Service, now func() time.Time) { s.opt.Now = now }

// SetKidsTimers encurta o vigia do perfil infantil.
func SetKidsTimers(tick, warn time.Duration) { kidsTick, kidsWarn = tick, warn }
