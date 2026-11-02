package launcher

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"github.com/stanleygomes/arkflix-games/app/internal/catalog"
)

var names = map[string]string{catalog.RetroArch: "RetroArch", catalog.PPSSPP: "PPSSPP", catalog.PCSX2: "PCSX2"}

var sites = map[string]string{
	catalog.RetroArch: "retroarch.com",
	catalog.PPSSPP:    "ppsspp.org",
	catalog.PCSX2:     "pcsx2.net",
}

func (s Spec) goos() string {
	if s.GOOS != "" {
		return s.GOOS
	}
	return runtime.GOOS
}

// CoreExt: extensão dos cores do RetroArch em cada sistema.
func CoreExt(goos string) string {
	switch goos {
	case "windows":
		return ".dll"
	case "darwin":
		return ".dylib"
	}
	return ".so"
}

// Candidates: onde os instaladores oficiais põem cada emulador.
func Candidates(goos, emu, home string, env func(string) string) []string {
	switch goos {
	case "windows":
		pf, local := env("ProgramFiles"), env("LOCALAPPDATA")
		if pf == "" {
			pf = `C:\Program Files`
		}
		switch emu {
		case catalog.RetroArch:
			return []string{`C:\RetroArch-Win64\retroarch.exe`, filepath.Join(pf, "RetroArch", "retroarch.exe"),
				filepath.Join(local, "Programs", "RetroArch", "retroarch.exe"), `C:\Program Files (x86)\Steam\steamapps\common\RetroArch\retroarch.exe`}
		case catalog.PPSSPP:
			return []string{filepath.Join(pf, "PPSSPP", "PPSSPPWindows64.exe"), filepath.Join(local, "Programs", "PPSSPP", "PPSSPPWindows64.exe")}
		case catalog.PCSX2:
			return []string{filepath.Join(pf, "PCSX2", "pcsx2-qt.exe"), filepath.Join(local, "Programs", "PCSX2", "pcsx2-qt.exe")}
		}
	case "darwin":
		app := map[string]string{
			catalog.RetroArch: "RetroArch.app/Contents/MacOS/RetroArch",
			catalog.PPSSPP:    "PPSSPPSDL.app/Contents/MacOS/PPSSPPSDL",
			catalog.PCSX2:     "PCSX2.app/Contents/MacOS/PCSX2",
		}[emu]
		return []string{filepath.Join("/Applications", app), filepath.Join(home, "Applications", app)}
	}
	return nil
}

// pathNames: nomes pra procurar no PATH quando não está no lugar padrão.
var pathNames = map[string][]string{
	catalog.RetroArch: {"retroarch"},
	catalog.PPSSPP:    {"PPSSPPWindows64", "PPSSPPSDL", "ppsspp"},
	catalog.PCSX2:     {"pcsx2-qt", "pcsx2"},
}

// Find acha o emulador instalado (vazio se não achar).
func Find(goos, emu string) string {
	home, _ := os.UserHomeDir()
	for _, p := range Candidates(goos, emu, home, os.Getenv) {
		if st, err := os.Stat(p); err == nil && !st.IsDir() {
			return p
		}
	}
	for _, n := range pathNames[emu] {
		if p, err := exec.LookPath(n); err == nil {
			return p
		}
	}
	return ""
}

// DefaultCores: pasta de cores do RetroArch (raExe = executável em uso).
func DefaultCores(goos, raExe string) string {
	switch goos {
	case "windows":
		return filepath.Join(filepath.Dir(raExe), "cores")
	case "darwin":
		home, _ := os.UserHomeDir()
		return filepath.Join(home, "Library", "Application Support", "RetroArch", "cores")
	}
	return FlatpakCores()
}
