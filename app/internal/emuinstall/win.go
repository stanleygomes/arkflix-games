package emuinstall

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/stanleygomes/arkflix-games/app/internal/catalog"
	"github.com/stanleygomes/arkflix-games/app/internal/launcher"
)

// winExe: o executável de cada emulador nos pacotes oficiais do Windows.
var winExe = map[string]string{
	catalog.RetroArch: "retroarch.exe",
	catalog.PPSSPP:    "PPSSPPWindows64.exe",
	catalog.PCSX2:     "pcsx2-qt.exe",
}

// winTarget: onde o app instala, %LOCALAPPDATA%\Programs\<Nome> (um dos
// lugares que o launcher procura; não pede administrador).
func (in *Installer) winTarget(emu string) string {
	local := in.env("LOCALAPPDATA")
	if local == "" {
		local = filepath.Join(in.Home, "AppData", "Local")
	}
	return filepath.Join(local, "Programs", names[emu])
}

// winURL: o pacote portátil oficial de cada emulador.
func (in *Installer) winURL(ctx context.Context, emu string) (string, error) {
	switch emu {
	case catalog.RetroArch:
		return fmt.Sprintf("%s/stable/%s/windows/x86_64/RetroArch.7z", in.Buildbot, in.retroArchVersion(ctx)), nil
	case catalog.PPSSPP:
		return fmt.Sprintf("%s/%s/ppsspp_win.zip", in.PPSSPP, strings.ReplaceAll(PPSSPPVersion, ".", "_")), nil
	}
	// o "-symbols.7z" também termina em .7z: o sufixo inteiro separa
	return in.githubAsset(ctx, "PCSX2/pcsx2", "windows-x64-Qt.7z")
}

// winInstall baixa e abre o pacote por cima da instalação atual (a do
// RetroArch é portátil: config, saves e cores moram na pasta e ficam).
func (in *Installer) winInstall(ctx context.Context, emu string, progress func(float64)) error {
	dst := in.winTarget(emu)
	if exe := in.find(emu); exe != "" {
		dst = filepath.Dir(exe) // atualiza onde já está
	}
	if err := os.MkdirAll(dst, 0o755); err != nil {
		return fmt.Errorf("%s: %w", names[emu], err)
	}
	url, err := in.winURL(ctx, emu)
	if err != nil {
		return fmt.Errorf("%s: %w", names[emu], err)
	}
	file, err := in.download(ctx, url, dst, maxArchive, func(f float64) { progress(f * 0.7) })
	if err != nil {
		return fmt.Errorf("%s: %w", names[emu], err)
	}
	defer os.Remove(file)
	ext := filepath.Ext(url) // openArchive escolhe zip ou 7z pela extensão
	if err := os.Rename(file, file+ext); err != nil {
		return err
	}
	file += ext
	defer os.Remove(file)
	if err := extractTree(file, winExe[emu], dst); err != nil {
		return fmt.Errorf("%s: %w", names[emu], err)
	}
	if emu == catalog.RetroArch {
		os.MkdirAll(launcher.DefaultCores("windows", filepath.Join(dst, winExe[emu])), 0o755)
	}
	progress(1)
	return nil
}
