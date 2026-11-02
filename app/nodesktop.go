//go:build !desktop

package main

import (
	"io/fs"
	"log"

	"github.com/stanleygomes/arkflix-games/app/internal/bt"
	"github.com/stanleygomes/arkflix-games/app/internal/gamepad"
	"github.com/stanleygomes/arkflix-games/app/internal/launcher"
	"github.com/stanleygomes/arkflix-games/app/internal/settings"
)

// Sem a tag desktop (testes, modo web) não há janela: use --web.
func runDesktop(*settings.Store, *gamepad.Manager, fs.FS, bool, bool, launcher.Spec, bt.Backend) int {
	log.Print("compilado sem janela (tag desktop): use --web 127.0.0.1:8484")
	return 2
}
