package emuinstall

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/stanleygomes/arkflix-games/app/internal/catalog"
	"github.com/stanleygomes/arkflix-games/app/internal/launcher"
)

// flatpakApp: um app instalado (flatpak list).
type flatpakApp struct {
	version, installation string
}

// flatpakList: os apps instalados, de usuário e de sistema, numa chamada só.
func (in *Installer) flatpakList(ctx context.Context) map[string]flatpakApp {
	ctx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()
	out, err := in.Run(ctx, "flatpak", "list", "--app", "--columns=application,version,installation")
	apps := map[string]flatpakApp{}
	if err != nil {
		return apps
	}
	for _, line := range strings.Split(string(out), "\n") {
		f := strings.Split(line, "\t")
		if len(f) >= 3 {
			apps[strings.TrimSpace(f[0])] = flatpakApp{version: strings.TrimSpace(f[1]), installation: strings.TrimSpace(f[2])}
		}
	}
	return apps
}

func (in *Installer) flatpakStatus(ctx context.Context) []Emulator {
	apps := in.flatpakList(ctx)
	var out []Emulator
	for _, emu := range emulators {
		id := launcher.Flatpaks[emu]
		app, ok := apps[id]
		out = append(out, Emulator{ID: emu, Name: names[emu], Installed: ok, Path: id, Version: app.version})
	}
	return out
}

// flathubRemote garante o repositório do Flathub (só do usuário: sem sudo).
func (in *Installer) flathubRemote(ctx context.Context, _ func(float64)) error {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()
	if _, err := in.Run(ctx, "flatpak", "remote-add", "--user", "--if-not-exists", "flathub", in.Flathub); err != nil {
		return fmt.Errorf("não deu pra adicionar o Flathub: %w", err)
	}
	return nil
}

// flatpakInstall instala (ou atualiza, se já tem) o Flatpak do emulador.
func (in *Installer) flatpakInstall(ctx context.Context, emu string, installed bool) error {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Minute)
	defer cancel()
	id := launcher.Flatpaks[emu]
	args := []string{"install", "--user", "-y", "--noninteractive", "flathub", id}
	if installed {
		args = []string{"update", "-y", "--noninteractive", id}
		if in.flatpakList(ctx)[id].installation == "user" {
			args = []string{"update", "--user", "-y", "--noninteractive", id}
		}
	}
	if _, err := in.Run(ctx, "flatpak", args...); err != nil {
		return fmt.Errorf("%s: %w", names[emu], err)
	}
	if emu == catalog.PCSX2 {
		// o PCSX2 do Flathub não enxerga a home: libera leitura pros jogos de PS2
		if _, err := in.Run(ctx, "flatpak", "override", "--user", "--filesystem=host:ro", id); err != nil {
			return fmt.Errorf("%s: %w", names[emu], err)
		}
	}
	return nil
}
