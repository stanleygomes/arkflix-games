package emuinstall

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/stanleygomes/arkflix-games/app/internal/catalog"
)

// macApp: o nome do .app que o launcher procura (o PCSX2 vem como
// PCSX2-v2.8.2.app e é renomeado).
var macApp = map[string]string{
	catalog.RetroArch: "RetroArch.app",
	catalog.PPSSPP:    "PPSSPPSDL.app",
	catalog.PCSX2:     "PCSX2.app",
}

// macURL: o .dmg (RetroArch, PPSSPP) ou .tar.xz (PCSX2) oficial.
func (in *Installer) macURL(ctx context.Context, emu string) (string, error) {
	switch emu {
	case catalog.RetroArch:
		return fmt.Sprintf("%s/stable/%s/apple/osx/universal/RetroArch_Metal.dmg", in.Buildbot, in.retroArchVersion(ctx)), nil
	case catalog.PPSSPP:
		return fmt.Sprintf("%s/%s/PPSSPP_macOS.dmg", in.PPSSPP, strings.ReplaceAll(PPSSPPVersion, ".", "_")), nil
	}
	return in.githubAsset(ctx, "PCSX2/pcsx2", "macos-Qt.tar.xz")
}

// appDir: a pasta .app de um executável .app/Contents/MacOS/X.
func appDir(exe string) string {
	return filepath.Dir(filepath.Dir(filepath.Dir(exe)))
}

// macTarget: onde já está (atualiza no lugar) ou ~/Applications.
func (in *Installer) macTarget(emu string) string {
	if exe := in.find(emu); exe != "" {
		return appDir(exe)
	}
	return filepath.Join(in.Home, "Applications", macApp[emu])
}

// macInstall baixa, abre (hdiutil ou tar) e troca o .app inteiro.
func (in *Installer) macInstall(ctx context.Context, emu string, progress func(float64)) error {
	err := in.macInstallApp(ctx, emu, progress)
	if err != nil {
		return fmt.Errorf("%s: %w", names[emu], err)
	}
	return nil
}

func (in *Installer) macInstallApp(ctx context.Context, emu string, progress func(float64)) error {
	url, err := in.macURL(ctx, emu)
	if err != nil {
		return err
	}
	work, err := os.MkdirTemp("", "arkflix-emu-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(work)
	file, err := in.download(ctx, url, work, maxArchive, func(f float64) { progress(f * 0.7) })
	if err != nil {
		return err
	}
	src := filepath.Join(work, "src")
	if err := os.Mkdir(src, 0o755); err != nil {
		return err
	}
	if strings.HasSuffix(url, ".dmg") {
		if _, err := in.Run(ctx, "hdiutil", "attach", "-nobrowse", "-readonly", "-noautoopen", "-mountpoint", src, file); err != nil {
			return fmt.Errorf("não deu pra abrir o .dmg: %w", err)
		}
		defer func() {
			// desmonta mesmo se o ctx já acabou
			dctx, cancel := context.WithTimeout(context.Background(), time.Minute)
			defer cancel()
			in.Run(dctx, "hdiutil", "detach", "-force", src)
		}()
	} else if _, err := in.Run(ctx, "tar", "-xf", file, "-C", src); err != nil {
		return fmt.Errorf("não deu pra abrir o pacote: %w", err)
	}
	app, err := findApp(src)
	if err != nil {
		return err
	}
	progress(0.8)
	return in.swapApp(ctx, app, in.macTarget(emu))
}

// findApp: o primeiro .app na raiz (o atalho "Applications" do dmg é link e fica de fora).
func findApp(dir string) (string, error) {
	items, err := os.ReadDir(dir)
	if err != nil {
		return "", err
	}
	for _, it := range items {
		if it.IsDir() && strings.HasSuffix(it.Name(), ".app") {
			return filepath.Join(dir, it.Name()), nil
		}
	}
	return "", errors.New("nenhum .app no pacote")
}

// swapApp copia com ditto (mantém assinatura e atributos) pra um nome
// temporário e só então troca pelo antigo.
func (in *Installer) swapApp(ctx context.Context, src, dst string) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	tmp, old := dst+".arkflix-new", dst+".arkflix-old"
	os.RemoveAll(tmp)
	os.RemoveAll(old)
	if _, err := in.Run(ctx, "ditto", src, tmp); err != nil {
		os.RemoveAll(tmp)
		return fmt.Errorf("não deu pra copiar o app: %w", err)
	}
	had := false
	if _, err := os.Stat(dst); err == nil {
		if err := os.Rename(dst, old); err != nil {
			os.RemoveAll(tmp)
			return fmt.Errorf("feche o emulador e tente de novo: %w", err)
		}
		had = true
	}
	if err := os.Rename(tmp, dst); err != nil {
		if had {
			os.Rename(old, dst)
		}
		return err
	}
	os.RemoveAll(old)
	return nil
}

var plistRe = regexp.MustCompile(`<key>CFBundleShortVersionString</key>\s*<string>([^<]+)</string>`)

// plistVersion: a versão do Info.plist do .app (vazio se não achar).
func plistVersion(app string) string {
	data, err := os.ReadFile(filepath.Join(app, "Contents", "Info.plist"))
	if err != nil {
		return ""
	}
	if m := plistRe.FindSubmatch(data); m != nil {
		return strings.TrimSpace(string(m[1]))
	}
	return ""
}
