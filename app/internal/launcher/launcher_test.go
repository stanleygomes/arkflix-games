package launcher_test

import (
	"context"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/stanleygomes/arkflix-games/app/internal/catalog"
	"github.com/stanleygomes/arkflix-games/app/internal/launcher"
)

func setup(t *testing.T) launcher.Spec {
	t.Helper()
	bin, cores := t.TempDir(), t.TempDir()
	for _, name := range []string{"retroarch", "ppsspp", "pcsx2"} {
		script := "#!/bin/sh\necho \"$0 $*\" > \"$(dirname \"$0\")/args.txt\"\n[ \"$FAIL\" = 1 ] && { echo boom >&2; exit 3; }\nexit 0\n"
		os.WriteFile(filepath.Join(bin, name), []byte(script), 0o755)
	}
	os.WriteFile(filepath.Join(cores, "snes9x_libretro.so"), nil, 0o644)
	return launcher.Spec{Bin: bin, Cores: cores, Fullscreen: true, RAAppend: []string{"/x/input.cfg"}, Work: t.TempDir()}
}

func TestRetroArch(t *testing.T) {
	spec := setup(t)
	snes, _ := catalog.Find("snes")
	args, err := launcher.Command(snes, "/roms/smw.zip", spec, &launcher.Resume{Slot: 3})
	if err != nil {
		t.Fatal(err)
	}
	want := []string{filepath.Join(spec.Bin, "retroarch"), "--fullscreen", "--entryslot=3", "--appendconfig=/x/input.cfg", "-L", filepath.Join(spec.Cores, "snes9x_libretro.so"), "/roms/smw.zip"}
	if !slices.Equal(args, want) {
		t.Fatalf("args %v\nwant %v", args, want)
	}
	args, _ = launcher.Command(snes, "/roms/smw.zip", spec, &launcher.Resume{Slot: -1})
	auto := filepath.Join(spec.Work, "arkflix-resume.cfg")
	if !slices.Contains(args, "--appendconfig=/x/input.cfg|"+auto) {
		t.Fatalf("continuar devia anexar config de autoload na pasta de trabalho: %v", args)
	}
	if data, _ := os.ReadFile(auto); !strings.Contains(string(data), "savestate_auto_load") {
		t.Fatalf("autoload: %s", data)
	}
}

func TestMissingCore(t *testing.T) {
	spec := setup(t)
	nes, _ := catalog.Find("nes")
	if _, err := launcher.Command(nes, "/roms/smb.zip", spec, nil); err == nil || !strings.Contains(err.Error(), "make emulators") {
		t.Fatalf("core ausente devia dizer como instalar: %v", err)
	}
}

func TestStandalone(t *testing.T) {
	spec := setup(t)
	psp, _ := catalog.Find("psp")
	args, err := launcher.Command(psp, "/roms/lara.iso", spec, &launcher.Resume{File: "/s/ULUS_1.00_0.ppst"})
	if err != nil || !slices.Contains(args, "--state=/s/ULUS_1.00_0.ppst") || args[len(args)-1] != "/roms/lara.iso" {
		t.Fatalf("psp: %v %v", args, err)
	}
	ps2, _ := catalog.Find("ps2")
	args, err = launcher.Command(ps2, "/roms/-fifa.iso", spec, &launcher.Resume{File: "/s/x.p2s"})
	if err != nil || !slices.Equal(args[1:], []string{"-batch", "-fullscreen", "-statefile", "/s/x.p2s", "--", "/roms/-fifa.iso"}) {
		t.Fatalf("ps2: %v %v", args, err)
	}
}

// Sem Bin: usa "flatpak run <id>" (com um flatpak falso no PATH).
func TestFlatpak(t *testing.T) {
	fake := t.TempDir()
	os.WriteFile(filepath.Join(fake, "flatpak"), []byte("#!/bin/sh\n[ \"$1\" = info ] && [ \"$3\" != net.pcsx2.PCSX2 ]\n"), 0o755)
	t.Setenv("PATH", fake)
	cores := t.TempDir()
	os.WriteFile(filepath.Join(cores, "snes9x_libretro.so"), nil, 0o644)
	snes, _ := catalog.Find("snes")
	args, err := launcher.Command(snes, "/roms/smw.zip", launcher.Spec{Cores: cores}, nil)
	if err != nil || !slices.Equal(args[:3], []string{"flatpak", "run", "org.libretro.RetroArch"}) {
		t.Fatalf("retroarch pelo flatpak: %v %v", args, err)
	}
	ps2, _ := catalog.Find("ps2")
	if _, err := launcher.Command(ps2, "/roms/fifa.iso", launcher.Spec{}, nil); err == nil || !strings.Contains(err.Error(), "net.pcsx2.PCSX2 não instalado") {
		t.Fatalf("flatpak ausente: %v", err)
	}
	t.Setenv("PATH", t.TempDir())
	if _, err := launcher.Command(snes, "/roms/smw.zip", launcher.Spec{Cores: cores}, nil); err == nil || !strings.Contains(err.Error(), "falta o Flatpak") {
		t.Fatalf("sem flatpak: %v", err)
	}
	if !strings.HasSuffix(launcher.FlatpakCores(), ".var/app/org.libretro.RetroArch/config/retroarch/cores") {
		t.Fatal(launcher.FlatpakCores())
	}
}

func TestRun(t *testing.T) {
	spec := setup(t)
	snes, _ := catalog.Find("snes")
	args, _ := launcher.Command(snes, "/roms/smw.zip", spec, nil)
	res, err := launcher.Run(context.Background(), args, nil)
	if err != nil || res.Code != 0 {
		t.Fatalf("run: %+v %v", res, err)
	}
	got, _ := os.ReadFile(filepath.Join(spec.Bin, "args.txt"))
	if !strings.Contains(string(got), "/roms/smw.zip") {
		t.Fatalf("lançador não recebeu o jogo: %s", got)
	}
	res, err = launcher.Run(context.Background(), args, []string{"FAIL=1"})
	if err != nil || res.Code != 3 || !strings.Contains(res.Tail, "boom") {
		t.Fatalf("falha devia trazer código e saída: %+v %v", res, err)
	}
}

func TestPlatforms(t *testing.T) {
	if launcher.CoreExt("windows") != ".dll" || launcher.CoreExt("darwin") != ".dylib" || launcher.CoreExt("linux") != ".so" {
		t.Fatal("extensão dos cores por sistema")
	}
	env := func(k string) string {
		return map[string]string{"ProgramFiles": `D:\Apps`, "LOCALAPPDATA": `C:\Users\s\AppData\Local`}[k]
	}
	win := launcher.Candidates("windows", catalog.RetroArch, `C:\Users\s`, env)
	if win[0] != `C:\RetroArch-Win64\retroarch.exe` || !strings.Contains(strings.Join(win, "|"), "Apps") {
		t.Fatalf("windows: %v", win)
	}
	if ps2 := launcher.Candidates("windows", catalog.PCSX2, "", env); !strings.HasSuffix(ps2[0], "pcsx2-qt.exe") {
		t.Fatalf("pcsx2 windows: %v", ps2)
	}
	mac := launcher.Candidates("darwin", catalog.PPSSPP, "/Users/s", env)
	if mac[0] != "/Applications/PPSSPPSDL.app/Contents/MacOS/PPSSPPSDL" || mac[1] != "/Users/s/Applications/PPSSPPSDL.app/Contents/MacOS/PPSSPPSDL" {
		t.Fatalf("mac: %v", mac)
	}
	if launcher.Candidates("linux", catalog.RetroArch, "/h", env) != nil {
		t.Fatal("linux usa Flatpak, sem lista")
	}
}

// Windows/macOS: caminho escolhido nas configurações, ou erro dizendo onde baixar.
func TestNativeExe(t *testing.T) {
	dir := t.TempDir()
	exe := filepath.Join(dir, "retroarch.exe")
	os.WriteFile(exe, []byte("x"), 0o755)
	os.MkdirAll(filepath.Join(dir, "cores"), 0o755)
	os.WriteFile(filepath.Join(dir, "cores", "snes9x_libretro.dll"), nil, 0o644)
	snes, _ := catalog.Find("snes")
	spec := launcher.Spec{GOOS: "windows", Exe: map[string]string{catalog.RetroArch: exe}}
	args, err := launcher.Command(snes, `C:\roms\smw.zip`, spec, nil)
	if err != nil || args[0] != exe || !slices.Contains(args, filepath.Join(dir, "cores", "snes9x_libretro.dll")) {
		t.Fatalf("retroarch.exe com cores ao lado: %v %v", args, err)
	}
	t.Setenv("PATH", t.TempDir())
	psp, _ := catalog.Find("psp")
	if _, err := launcher.Command(psp, "/r/x.iso", launcher.Spec{GOOS: "darwin"}, nil); err == nil || !strings.Contains(err.Error(), "ppsspp.org") {
		t.Fatalf("sem PPSSPP: diz onde baixar: %v", err)
	}
	spec.Exe[catalog.RetroArch] = filepath.Join(dir, "sumiu.exe")
	if _, err := launcher.Command(snes, "x.zip", spec, nil); err == nil || !strings.Contains(err.Error(), "configurações") {
		t.Fatalf("caminho escolhido sumiu: %v", err)
	}
}

func TestGameCoreAndShader(t *testing.T) {
	spec := setup(t)
	os.WriteFile(filepath.Join(spec.Cores, "bsnes_libretro.so"), nil, 0o644)
	snes, _ := catalog.Find("snes")
	spec.Core, spec.Shader = "bsnes", "/sh/crt-easymode.slangp"
	args, err := launcher.Command(snes, "/roms/smw.zip", spec, nil)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{filepath.Join(spec.Bin, "retroarch"), "--fullscreen", "--appendconfig=/x/input.cfg",
		"--set-shader", "/sh/crt-easymode.slangp", "-L", filepath.Join(spec.Cores, "bsnes_libretro.so"), "/roms/smw.zip"}
	if !slices.Equal(args, want) {
		t.Fatalf("args %v\nwant %v", args, want)
	}
	spec.Core = "../../etc/evil"
	if _, err := launcher.Command(snes, "/roms/smw.zip", spec, nil); err == nil {
		t.Fatal("core fora da lista do console")
	}
	spec.Core = "nestopia"
	if _, err := launcher.Command(snes, "/roms/smw.zip", spec, nil); err == nil {
		t.Fatal("core de outro console")
	}
}
