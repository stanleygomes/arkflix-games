package emucfg_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stanleygomes/arkflix-games/app/internal/emucfg"
)

const controls = "\uFEFF[ControlMapping]\r\nUp = 1-19,10-19\r\nStart = 1-41,10-197\r\nSelect = 1-66,10-196\r\nTriangle = 1-51,10-188\r\nPause = 1-111,10-109,10-104\r\nRewind = 1-67\r\n"

const general = "[General]\nFirstRun = False\nPauseOnLostFocus = False\n\n[CPU]\nFastMemoryAccess = True\n"

func TestPPSSPP(t *testing.T) {
	dir := t.TempDir()
	if err := emucfg.PPSSPP(dir); err != emucfg.ErrNotSetUp {
		t.Fatalf("sem config ainda: %v", err)
	}
	os.WriteFile(filepath.Join(dir, "controls.ini"), []byte(controls), 0o644)
	os.WriteFile(filepath.Join(dir, "ppsspp.ini"), []byte(general), 0o644)
	if err := emucfg.PPSSPP(dir); err != nil {
		t.Fatal(err)
	}
	got, _ := os.ReadFile(filepath.Join(dir, "controls.ini"))
	s := string(got)
	for _, want := range []string{"\uFEFF[ControlMapping]", "Pause = 1-111,10-196:10-188\n", "Exit App = 1-140,10-196:10-197\n", "Up = 1-19,10-19\n", "Rewind = 1-67\n"} {
		if !strings.Contains(s, want) {
			t.Errorf("controls.ini sem %q:\n%s", want, s)
		}
	}
	if strings.Contains(s, "10-109") {
		t.Error("Select sozinho não pode mais pausar (vira atalho com outro botão)")
	}
	g, _ := os.ReadFile(filepath.Join(dir, "ppsspp.ini"))
	if !strings.Contains(string(g), "[General]\nFirstRun = False\nPauseOnLostFocus = False\nPauseMenuExitsEmulator = True\n\n[CPU]") {
		t.Errorf("ppsspp.ini:\n%s", g)
	}
	if b, _ := os.ReadFile(filepath.Join(dir, "controls.ini.arkflix-backup")); string(b) != controls {
		t.Error("backup do original")
	}
	// de novo: nada muda, e o backup continua sendo o original
	before, _ := os.ReadFile(filepath.Join(dir, "controls.ini"))
	if err := emucfg.PPSSPP(dir); err != nil {
		t.Fatal(err)
	}
	after, _ := os.ReadFile(filepath.Join(dir, "controls.ini"))
	if string(before) != string(after) {
		t.Error("rodar de novo não pode mudar nada")
	}
}

func TestPPSSPPKeyboardOnly(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "controls.ini"), []byte("[ControlMapping]\nStart = 1-41\n"), 0o644)
	os.WriteFile(filepath.Join(dir, "ppsspp.ini"), []byte("[General]\n"), 0o644)
	if err := emucfg.PPSSPP(dir); err != nil {
		t.Fatal(err)
	}
	got, _ := os.ReadFile(filepath.Join(dir, "controls.ini"))
	if !strings.Contains(string(got), "Pause = 1-111\n") || !strings.Contains(string(got), "Exit App = 1-140\n") {
		t.Fatalf("sem controle mapeado, só teclado:\n%s", got)
	}
}

func TestPCSX2(t *testing.T) {
	ini := filepath.Join(t.TempDir(), "inis", "PCSX2.ini")
	if err := emucfg.PCSX2(ini); err != emucfg.ErrNotSetUp {
		t.Fatalf("antes do primeiro uso (assistente do PCSX2 cria o arquivo): %v", err)
	}
	os.MkdirAll(filepath.Dir(ini), 0o755)
	os.WriteFile(ini, []byte("[UI]\nSettingsVersion = 1\nConfirmShutdown = true\n\n[Hotkeys]\nOpenPauseMenu = Keyboard/Escape\nToggleFullscreen = Keyboard/Alt & Keyboard/Return\n"), 0o644)
	if err := emucfg.PCSX2(ini); err != nil {
		t.Fatal(err)
	}
	got, _ := os.ReadFile(ini)
	want := "[UI]\nSettingsVersion = 1\nConfirmShutdown = false\n\n[Hotkeys]\nOpenPauseMenu = Keyboard/Escape\nOpenPauseMenu = SDL-0/Back & SDL-0/Y\nToggleFullscreen = Keyboard/Alt & Keyboard/Return\nShutdownVM = Keyboard/F10\nShutdownVM = SDL-0/Back & SDL-0/Start\n"
	if string(got) != want {
		t.Fatalf("PCSX2.ini:\n%s\nesperado:\n%s", got, want)
	}
}

func TestIniGetSet(t *testing.T) {
	path := filepath.Join(t.TempDir(), "x.ini")
	ini, _ := emucfg.ReadIni(path)
	ini.Set("A", "k", "1")
	ini.Set("B", "k", "2", "3")
	ini.Set("A", "k", "4")
	if got := ini.Get("B", "k"); len(got) != 2 || got[1] != "3" {
		t.Fatalf("lista: %v", got)
	}
	if ini.String() != "[A]\nk = 4\n\n[B]\nk = 2\nk = 3\n" {
		t.Fatalf("%q", ini.String())
	}
}
