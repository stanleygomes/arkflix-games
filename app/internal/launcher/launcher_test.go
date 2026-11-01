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
		script := "#!/bin/sh\necho \"$0 $*\" > \"$(dirname \"$0\")/args.txt\"\necho \"$ARKFLIX_RA_APPEND\" >> \"$(dirname \"$0\")/args.txt\"\n[ \"$FAIL\" = 1 ] && { echo boom >&2; exit 3; }\nexit 0\n"
		os.WriteFile(filepath.Join(bin, name), []byte(script), 0o755)
	}
	os.WriteFile(filepath.Join(cores, "snes9x_libretro.so"), nil, 0o644)
	return launcher.Spec{Bin: bin, Cores: cores, Fullscreen: true, RAAppend: []string{"/x/input.cfg"}}
}

func TestRetroArch(t *testing.T) {
	spec := setup(t)
	snes, _ := catalog.Find("snes")
	args, env, err := launcher.Command(snes, "/roms/smw.zip", spec, &launcher.Resume{Slot: 3})
	if err != nil {
		t.Fatal(err)
	}
	want := []string{filepath.Join(spec.Bin, "retroarch"), "--fullscreen", "--entryslot=3", "-L", filepath.Join(spec.Cores, "snes9x_libretro.so"), "/roms/smw.zip"}
	if !slices.Equal(args, want) {
		t.Fatalf("args %v\nwant %v", args, want)
	}
	if len(env) != 1 || env[0] != "ARKFLIX_RA_APPEND=/x/input.cfg" {
		t.Fatalf("env %v", env)
	}
	_, env, _ = launcher.Command(snes, "/roms/smw.zip", spec, &launcher.Resume{Slot: -1})
	if !strings.Contains(env[0], "/x/input.cfg|") || !strings.Contains(env[0], "arkflix-resume.cfg") {
		t.Fatalf("continuar devia anexar config de autoload: %v", env)
	}
}

func TestMissingCore(t *testing.T) {
	spec := setup(t)
	nes, _ := catalog.Find("nes")
	if _, _, err := launcher.Command(nes, "/roms/smb.zip", spec, nil); err == nil || !strings.Contains(err.Error(), "make core-mesen") {
		t.Fatalf("core ausente devia dizer como compilar: %v", err)
	}
}

func TestStandalone(t *testing.T) {
	spec := setup(t)
	psp, _ := catalog.Find("psp")
	args, _, err := launcher.Command(psp, "/roms/lara.iso", spec, &launcher.Resume{File: "/s/ULUS_1.00_0.ppst"})
	if err != nil || !slices.Contains(args, "--state=/s/ULUS_1.00_0.ppst") || args[len(args)-1] != "/roms/lara.iso" {
		t.Fatalf("psp: %v %v", args, err)
	}
	ps2, _ := catalog.Find("ps2")
	args, _, err = launcher.Command(ps2, "/roms/-fifa.iso", spec, &launcher.Resume{File: "/s/x.p2s"})
	if err != nil || !slices.Equal(args[1:], []string{"-batch", "-fullscreen", "-statefile", "/s/x.p2s", "--", "/roms/-fifa.iso"}) {
		t.Fatalf("ps2: %v %v", args, err)
	}
}

func TestRun(t *testing.T) {
	spec := setup(t)
	snes, _ := catalog.Find("snes")
	args, env, _ := launcher.Command(snes, "/roms/smw.zip", spec, nil)
	res, err := launcher.Run(context.Background(), args, env)
	if err != nil || res.Code != 0 {
		t.Fatalf("run: %+v %v", res, err)
	}
	got, _ := os.ReadFile(filepath.Join(spec.Bin, "args.txt"))
	if !strings.Contains(string(got), "/roms/smw.zip") {
		t.Fatalf("lançador não recebeu o jogo: %s", got)
	}
	res, err = launcher.Run(context.Background(), args, append(env, "FAIL=1"))
	if err != nil || res.Code != 3 || !strings.Contains(res.Tail, "boom") {
		t.Fatalf("falha devia trazer código e saída: %+v %v", res, err)
	}
}
