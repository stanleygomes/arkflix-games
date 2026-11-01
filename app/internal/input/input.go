// Package input guarda o mapeamento de controle e teclado e gera a config de
// entrada do RetroArch. Os botões físicos usam nomes posicionais padrão
// (south = botão de baixo, east = da direita...), iguais em Xbox, PlayStation e
// 8BitDo. O RetroArch usa o driver SDL2, cujos índices seguem a mesma posição.
package input

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// Button é um botão do RetroPad (o controle virtual do RetroArch).
type Button struct {
	ID    string `json:"id"`
	Label string `json:"label"`
}

// RetroPad na ordem da tela de configuração.
var RetroPad = []Button{
	{"up", "Cima"}, {"down", "Baixo"}, {"left", "Esquerda"}, {"right", "Direita"},
	{"b", "B (baixo)"}, {"a", "A (direita)"}, {"y", "Y (esquerda)"}, {"x", "X (cima)"},
	{"l", "L"}, {"r", "R"}, {"l2", "L2"}, {"r2", "R2"},
	{"l3", "L3 (analógico esq.)"}, {"r3", "R3 (analógico dir.)"},
	{"select", "Select"}, {"start", "Start"},
}

// Physical são os botões físicos padrão e o índice SDL GameController de cada um.
// Gatilhos analógicos viram eixo: +4 (esquerdo) e +5 (direito).
var Physical = map[string]string{
	"south": "0", "east": "1", "west": "2", "north": "3",
	"select": "4", "home": "5", "start": "6", "l3": "7", "r3": "8",
	"l1": "9", "r1": "10", "up": "11", "down": "12", "left": "13", "right": "14",
	"l2": "+4", "r2": "+5",
}

// Mapping liga cada botão do RetroPad a um botão físico e a uma tecla.
type Mapping struct {
	Pad  map[string]string `json:"pad"`
	Keys map[string]string `json:"keys"`
}

// Default é o mapeamento posicional (igual ao SNES: B embaixo, A à direita).
func Default() Mapping {
	return Mapping{
		Pad: map[string]string{
			"up": "up", "down": "down", "left": "left", "right": "right",
			"b": "south", "a": "east", "y": "west", "x": "north",
			"l": "l1", "r": "r1", "l2": "l2", "r2": "r2", "l3": "l3", "r3": "r3",
			"select": "select", "start": "start",
		},
		Keys: map[string]string{
			"up": "up", "down": "down", "left": "left", "right": "right",
			"b": "z", "a": "x", "y": "a", "x": "s",
			"l": "q", "r": "w", "l2": "e", "r2": "r", "l3": "num1", "r3": "num3",
			"select": "rshift", "start": "enter",
		},
	}
}

// RetroArch só aceita nomes simples; isso também impede injetar linha na config.
var keyName = regexp.MustCompile(`^[a-z0-9_]{1,16}$`)

// Validate confere se todo botão existe e toda tecla tem nome válido.
func (m Mapping) Validate() error {
	ids := map[string]bool{}
	for _, b := range RetroPad {
		ids[b.ID] = true
	}
	for rp, phys := range m.Pad {
		if !ids[rp] {
			return fmt.Errorf("botão do RetroPad desconhecido: %s", rp)
		}
		if _, ok := Physical[phys]; !ok {
			return fmt.Errorf("botão físico desconhecido: %s", phys)
		}
	}
	for rp, key := range m.Keys {
		if !ids[rp] {
			return fmt.Errorf("botão do RetroPad desconhecido: %s", rp)
		}
		if !keyName.MatchString(key) {
			return fmt.Errorf("tecla inválida para %s: %q", rp, key)
		}
	}
	return nil
}

// Complete preenche o que faltar com o padrão.
func (m Mapping) Complete() Mapping {
	def := Default()
	out := Mapping{Pad: map[string]string{}, Keys: map[string]string{}}
	for _, b := range RetroPad {
		out.Pad[b.ID] = def.Pad[b.ID]
		if v, ok := m.Pad[b.ID]; ok {
			out.Pad[b.ID] = v
		}
		out.Keys[b.ID] = def.Keys[b.ID]
		if v, ok := m.Keys[b.ID]; ok {
			out.Keys[b.ID] = v
		}
	}
	return out
}

// Conflicts lista botões físicos usados por mais de um botão do RetroPad.
func (m Mapping) Conflicts() []string {
	used := map[string][]string{}
	for rp, phys := range m.Pad {
		used[phys] = append(used[phys], rp)
	}
	var out []string
	for phys, rps := range used {
		if len(rps) > 1 {
			sort.Strings(rps)
			out = append(out, fmt.Sprintf("%s: %s", phys, strings.Join(rps, ", ")))
		}
	}
	sort.Strings(out)
	return out
}

func bind(lines *[]string, prefix, phys string) {
	idx := Physical[phys]
	if strings.HasPrefix(idx, "+") {
		*lines = append(*lines, fmt.Sprintf("%s_btn = \"nul\"", prefix), fmt.Sprintf("%s_axis = \"%s\"", prefix, idx))
		return
	}
	*lines = append(*lines, fmt.Sprintf("%s_btn = \"%s\"", prefix, idx), fmt.Sprintf("%s_axis = \"nul\"", prefix))
}

// RetroArchConfig gera as linhas de entrada e atalhos do RetroArch.
// Atalhos no controle: Select segurado + Start sai, + R salva, + L carrega,
// + X (botão de cima) abre o menu. No teclado: F10, F2, F4, Esc. Iguais no
// PPSSPP e no PCSX2 (package emucfg).
//
// config_save_on_exit desligado: o RetroArch gravaria esta config (e as
// pastas de save de um perfil) por cima da config do Flatpak ao sair.
func RetroArchConfig(m Mapping) string {
	m = m.Complete()
	lines := []string{
		"# gerado pelo arkflix-games: não edite, use a tela de controles",
		`input_joypad_driver = "sdl2"`,
		`input_autodetect_enable = "true"`,
		`savestate_thumbnail_enable = "true"`,
		`savestate_auto_save = "true"`,
		`savestate_auto_index = "false"`,
		`quit_press_twice = "false"`,
		`pause_nonactive = "false"`,
		`config_save_on_exit = "false"`,
	}
	for _, b := range RetroPad {
		bind(&lines, "input_player1_"+b.ID, m.Pad[b.ID])
		lines = append(lines, fmt.Sprintf("input_player1_%s = \"%s\"", b.ID, m.Keys[b.ID]))
	}
	for _, axis := range []struct{ name, plus, minus string }{
		{"l_x", "+0", "-0"}, {"l_y", "+1", "-1"}, {"r_x", "+2", "-2"}, {"r_y", "+3", "-3"},
	} {
		lines = append(lines,
			fmt.Sprintf("input_player1_%s_plus_axis = \"%s\"", axis.name, axis.plus),
			fmt.Sprintf("input_player1_%s_minus_axis = \"%s\"", axis.name, axis.minus))
	}
	hotkeys := []struct{ name, rp, key string }{
		{"input_enable_hotkey", "select", "nul"},
		{"input_exit_emulator", "start", "f10"},
		{"input_save_state", "r", "f2"},
		{"input_load_state", "l", "f4"},
		{"input_menu_toggle", "x", "escape"},
	}
	for _, h := range hotkeys {
		bind(&lines, h.name, m.Pad[h.rp])
		lines = append(lines, fmt.Sprintf("%s = \"%s\"", h.name, h.key))
	}
	return strings.Join(lines, "\n") + "\n"
}

// Store guarda o mapeamento em input.json e a config do RetroArch ao lado.
type Store struct{ Dir string }

func (s Store) path() string { return filepath.Join(s.Dir, "input.json") }

// ConfigPath é o arquivo que o lançador do RetroArch anexa à config.
func (s Store) ConfigPath() string { return filepath.Join(s.Dir, "retroarch-input.cfg") }

// Load lê o mapeamento salvo (ou o padrão).
func (s Store) Load() (Mapping, error) {
	data, err := os.ReadFile(s.path())
	if errors.Is(err, os.ErrNotExist) {
		return Default(), nil
	}
	if err != nil {
		return Default(), err
	}
	var m Mapping
	if err := json.Unmarshal(data, &m); err != nil {
		return Default(), err
	}
	if err := m.Validate(); err != nil {
		return Default(), err
	}
	return m.Complete(), nil
}

// Save valida, grava e regenera a config do RetroArch.
func (s Store) Save(m Mapping) (Mapping, error) {
	if err := m.Validate(); err != nil {
		return Mapping{}, err
	}
	m = m.Complete()
	if err := os.MkdirAll(s.Dir, 0o700); err != nil {
		return Mapping{}, err
	}
	data, _ := json.MarshalIndent(m, "", "  ")
	if err := os.WriteFile(s.path(), data, 0o600); err != nil {
		return Mapping{}, err
	}
	return m, s.WriteConfig(m)
}

// WriteConfig grava retroarch-input.cfg.
func (s Store) WriteConfig(m Mapping) error {
	if err := os.MkdirAll(s.Dir, 0o700); err != nil {
		return err
	}
	return os.WriteFile(s.ConfigPath(), []byte(RetroArchConfig(m)), 0o644)
}
