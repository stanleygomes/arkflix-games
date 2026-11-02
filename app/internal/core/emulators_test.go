package core_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Com o fixture, a instalação é de mentira: só cria o que falta nas pastas de teste.
func TestInstallEmulatorsFixture(t *testing.T) {
	h := setup(t)
	os.Remove(filepath.Join(h.env.Bin, "ppsspp"))
	os.Remove(filepath.Join(h.env.Cores, "snes9x_libretro.so"))
	st := h.svc.EmulatorStatus()
	if !st.CanInstall || st.Busy || len(st.Emulators) != 3 || len(st.Cores) != 9 || st.Emulators[1].Installed || !st.Emulators[0].Installed {
		t.Fatalf("status: %+v", st)
	}
	if err := h.svc.InstallEmulators([]string{"dolphin"}); err == nil {
		t.Fatal("emulador desconhecido devia falhar")
	}
	if err := h.svc.InstallEmulators(nil); err != nil {
		t.Fatal(err)
	}
	if err := h.svc.InstallEmulators(nil); err == nil || !strings.Contains(err.Error(), "já estão") {
		t.Fatalf("duas ao mesmo tempo: %v", err)
	}
	ev := h.rec.waitTask(t, "emulators")
	if ev.State != "done" || ev.Message != "PPSSPP e 2 cores prontos" {
		t.Fatalf("fim: %+v", ev)
	}
	for _, f := range []string{filepath.Join(h.env.Bin, "ppsspp"), filepath.Join(h.env.Cores, "snes9x_libretro.so")} {
		if _, err := os.Stat(f); err != nil {
			t.Fatalf("faltou %s", f)
		}
	}
	st = h.svc.EmulatorStatus()
	if st.Busy || !st.Emulators[1].Installed {
		t.Fatalf("depois: %+v", st)
	}
	// o JSON que a interface lê: campos do Status no mesmo nível de busy
	data, _ := json.Marshal(st)
	for _, k := range []string{`"platform"`, `"canInstall"`, `"emulators"`, `"cores"`, `"coresDir"`, `"busy"`, `"installed"`} {
		if !strings.Contains(string(data), k) {
			t.Fatalf("json sem %s: %s", k, data)
		}
	}
}
