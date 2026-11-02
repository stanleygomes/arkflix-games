//go:build desktop

package main

import (
	"context"
	"io/fs"
	"log"
	"os/exec"
	goruntime "runtime"
	"sync"
	"time"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/linux"
	"github.com/wailsapp/wails/v2/pkg/runtime"

	"github.com/stanleygomes/arkflix-games/app/internal/bt"
	"github.com/stanleygomes/arkflix-games/app/internal/core"
	"github.com/stanleygomes/arkflix-games/app/internal/gamepad"
	"github.com/stanleygomes/arkflix-games/app/internal/launcher"
	"github.com/stanleygomes/arkflix-games/app/internal/settings"
)

// window liga o core à janela do Wails. Antes do OnStartup não há contexto:
// eventos e chamadas de janela são ignorados.
type window struct {
	mu  sync.Mutex
	ctx context.Context
}

func (w *window) get() context.Context {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.ctx
}

func (w *window) Emit(name string, data any) {
	if ctx := w.get(); ctx != nil {
		runtime.EventsEmit(ctx, name, data)
	}
}

func (w *window) Hide() {
	if ctx := w.get(); ctx != nil {
		runtime.WindowHide(ctx)
	}
}

func (w *window) Show() {
	if ctx := w.get(); ctx != nil {
		runtime.WindowShow(ctx)
	}
}

func (w *window) ToggleFullscreen() {
	if ctx := w.get(); ctx != nil {
		if runtime.WindowIsFullscreen(ctx) {
			runtime.WindowUnfullscreen(ctx)
		} else {
			runtime.WindowFullscreen(ctx)
		}
	}
}

func (w *window) Minimize() {
	if ctx := w.get(); ctx != nil {
		runtime.WindowMinimise(ctx)
	}
}

func (w *window) IsFullscreen() bool {
	if ctx := w.get(); ctx != nil {
		return runtime.WindowIsFullscreen(ctx)
	}
	return false
}

func (w *window) Quit() {
	if ctx := w.get(); ctx != nil {
		runtime.Quit(ctx)
	}
}

// OpenFile abre no programa padrão do sistema, sem esperar.
func (w *window) OpenFile(path string) error {
	switch goruntime.GOOS {
	case "windows":
		return exec.Command("rundll32", "url.dll,FileProtocolHandler", path).Start()
	case "darwin":
		return exec.Command("open", path).Start()
	}
	return exec.Command("xdg-open", path).Start()
}

func (w *window) PickFiles() ([]string, error) {
	ctx := w.get()
	if ctx == nil {
		return nil, nil
	}
	return runtime.OpenMultipleFilesDialog(ctx, runtime.OpenDialogOptions{
		Title: "Escolha ROMs pra importar",
		Filters: []runtime.FileFilter{
			{DisplayName: "ROMs e pacotes", Pattern: "*.zip;*.7z;*.rar;*.nes;*.sfc;*.smc;*.md;*.gen;*.bin;*.gb;*.gbc;*.z64;*.n64;*.v64;*.nds;*.cue;*.chd;*.pbp;*.m3u;*.iso;*.cso"},
			{DisplayName: "Todos os arquivos", Pattern: "*"},
		},
	})
}

func runDesktop(store *settings.Store, pads *gamepad.Manager, ui fs.FS, smoke, windowed bool, emu launcher.Spec, blue bt.Backend) int {
	// tela cheia por padrão (é um app de TV); F11 alterna, --windowed abre em janela
	start := options.Fullscreen
	if windowed {
		start = options.Maximised
	}
	win := &window{}
	svc, err := core.New(core.Options{Version: version, Mode: "desktop", Store: store, Pads: pads, Emitter: win, Window: win, MinPlay: minPlayDesktop, Cache: cacheDirDesktop(), Emulators: emu, Bluetooth: blue,
		Update: core.UpdateOptions{Check: !smoke}})
	if err != nil {
		log.Print(err)
		return 1
	}
	code := 0
	err = wails.Run(&options.App{
		Title:            "arkflix games",
		Width:            1280,
		Height:           800,
		MinWidth:         960,
		MinHeight:        600,
		WindowStartState: start,
		StartHidden:      smoke,
		BackgroundColour: &options.RGBA{R: 16, G: 14, B: 24, A: 255},
		AssetServer:      &assetserver.Options{Assets: ui, Handler: core.Media(svc)},
		Bind:             []any{svc},
		Linux:            &linux.Options{ProgramName: "arkflix-games"},
		// atualização baixada instala ao fechar (nunca com o app aberto)
		OnShutdown: func(context.Context) { core.ApplyUpdateOnQuit(svc) },
		OnStartup: func(ctx context.Context) {
			win.mu.Lock()
			win.ctx = ctx
			win.mu.Unlock()
			if smoke {
				go func() {
					if !svc.WaitReady(45 * time.Second) {
						log.Print("smoke: a interface não ficou pronta em 45 s")
						code = 1
					} else {
						log.Print("smoke: interface pronta")
					}
					runtime.Quit(ctx)
				}()
			}
		},
	})
	if err != nil {
		log.Print(err)
		return 1
	}
	return code
}
