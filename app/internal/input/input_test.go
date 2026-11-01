package input_test

import (
	"os"
	"strings"
	"testing"

	"github.com/stanleygomes/arkflix-games/app/internal/input"
)

func TestDefaultValid(t *testing.T) {
	m := input.Default()
	if err := m.Validate(); err != nil {
		t.Fatal(err)
	}
	if len(m.Conflicts()) != 0 {
		t.Fatalf("padrão com conflito: %v", m.Conflicts())
	}
	if len(m.Pad) != len(input.RetroPad) || len(m.Keys) != len(input.RetroPad) {
		t.Fatal("padrão não cobre todos os botões")
	}
}

func TestRetroArchConfig(t *testing.T) {
	cfg := input.RetroArchConfig(input.Default())
	for _, want := range []string{
		`input_joypad_driver = "sdl2"`,
		`input_player1_b_btn = "0"`, // south
		`input_player1_a_btn = "1"`, // east
		`input_player1_l2_axis = "+4"`,
		`input_player1_l2_btn = "nul"`,
		`input_player1_start = "enter"`,
		`input_enable_hotkey_btn = "4"`, // select
		`input_exit_emulator_btn = "6"`, // start
		`input_exit_emulator = "f10"`,
		`input_menu_toggle = "escape"`,
		`config_save_on_exit = "false"`, // não grava por cima da config do Flatpak
		`input_player1_l_x_plus_axis = "+0"`,
		`savestate_thumbnail_enable = "true"`,
	} {
		if !strings.Contains(cfg, want) {
			t.Errorf("config sem %s", want)
		}
	}
}

func TestRemapAndHotkeysFollow(t *testing.T) {
	m := input.Default()
	m.Pad["select"] = "home"
	cfg := input.RetroArchConfig(m)
	if !strings.Contains(cfg, `input_player1_select_btn = "5"`) || !strings.Contains(cfg, `input_enable_hotkey_btn = "5"`) {
		t.Fatal("atalho deve seguir o botão mapeado no Select")
	}
}

func TestValidateRejects(t *testing.T) {
	bad := []input.Mapping{
		{Pad: map[string]string{"zz": "south"}},
		{Pad: map[string]string{"a": "botao-magico"}},
		{Keys: map[string]string{"a": "x\"\ninput_exit_emulator = \"a"}},
		{Keys: map[string]string{"a": ""}},
	}
	for i, m := range bad {
		if err := m.Validate(); err == nil {
			t.Errorf("caso %d devia ser recusado", i)
		}
	}
}

func TestConflicts(t *testing.T) {
	m := input.Default()
	m.Pad["a"] = "south"
	if c := m.Conflicts(); len(c) != 1 || !strings.Contains(c[0], "a, b") {
		t.Fatalf("conflito não detectado: %v", c)
	}
}

func TestStore(t *testing.T) {
	s := input.Store{Dir: t.TempDir()}
	m, err := s.Load()
	if err != nil || m.Pad["a"] != "east" {
		t.Fatalf("load inicial: %v %v", m, err)
	}
	m.Keys["a"] = "k"
	if _, err := s.Save(input.Mapping{Keys: map[string]string{"a": "k"}}); err != nil {
		t.Fatal(err)
	}
	got, err := s.Load()
	if err != nil || got.Keys["a"] != "k" || got.Pad["b"] != "south" {
		t.Fatalf("salvo errado: %+v %v", got, err)
	}
	cfg, err := os.ReadFile(s.ConfigPath())
	if err != nil || !strings.Contains(string(cfg), `input_player1_a = "k"`) {
		t.Fatalf("config não regenerada: %v", err)
	}
}
