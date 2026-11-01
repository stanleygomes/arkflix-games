// Package catalog descreve os consoles suportados: nome, mídia, extensões,
// emulador e a paleta de cores que a interface usa quando o console está em foco.
package catalog

import (
	"path/filepath"
	"strings"
)

// Emuladores conhecidos. Cada um tem um lançador em build/bin.
const (
	RetroArch = "retroarch"
	PPSSPP    = "ppsspp"
	PCSX2     = "pcsx2"
)

// Palette é a identidade visual do console. Bg/Surface são fundos escuros;
// Accent é a cor de destaque (foco, botões); Text tem contraste AA sobre Bg.
type Palette struct {
	Bg      string `json:"bg"`
	Surface string `json:"surface"`
	Accent  string `json:"accent"`
	Accent2 string `json:"accent2"`
	Text    string `json:"text"`
	Muted   string `json:"muted"`
}

// System é um console. Slug é o mesmo nome de pasta usado no RomM e no ES-DE.
type System struct {
	Slug     string   `json:"slug"`
	Name     string   `json:"name"`
	Short    string   `json:"short"`
	Maker    string   `json:"maker"`
	Year     int      `json:"year"`
	Media    string   `json:"media"` // cartridge, disc, umd, card, arcade
	Exts     []string `json:"exts"`
	Emulator string   `json:"emulator"`
	Core     string   `json:"core,omitempty"`
	Palette  Palette  `json:"palette"`
}

const text = "#f4f1e8"

// Systems na ordem em que aparecem na barra de consoles.
var Systems = []System{
	{
		Slug: "nes", Name: "Nintendo Entertainment System", Short: "NES", Maker: "Nintendo",
		Year: 1983, Media: "cartridge", Exts: []string{".nes", ".unf", ".fds"},
		Emulator: RetroArch, Core: "mesen",
		Palette: Palette{Bg: "#17171b", Surface: "#26262c", Accent: "#ff4d5e", Accent2: "#b9bbc0", Text: text, Muted: "#b4b2ab"},
	},
	{
		Slug: "snes", Name: "Super Nintendo", Short: "SNES", Maker: "Nintendo",
		Year: 1990, Media: "cartridge", Exts: []string{".sfc", ".smc", ".swc", ".fig"},
		Emulator: RetroArch, Core: "snes9x",
		Palette: Palette{Bg: "#1b1728", Surface: "#2a2440", Accent: "#a99bff", Accent2: "#ffd25e", Text: text, Muted: "#bdb7cf"},
	},
	{
		Slug: "genesis", Name: "Mega Drive", Short: "MD", Maker: "Sega",
		Year: 1988, Media: "cartridge", Exts: []string{".md", ".gen", ".smd", ".bin", ".sms", ".gg"},
		Emulator: RetroArch, Core: "genesis_plus_gx",
		Palette: Palette{Bg: "#0f0f12", Surface: "#1f1f25", Accent: "#ff5a4f", Accent2: "#6f93ff", Text: text, Muted: "#b1aea7"},
	},
	{
		Slug: "gbc", Name: "Game Boy Color", Short: "GBC", Maker: "Nintendo",
		Year: 1998, Media: "cartridge", Exts: []string{".gb", ".gbc", ".sgb"},
		Emulator: RetroArch, Core: "gambatte",
		Palette: Palette{Bg: "#0e1a11", Surface: "#1b2e1f", Accent: "#b4d84a", Accent2: "#7fb3d9", Text: text, Muted: "#b6c2ae"},
	},
	{
		Slug: "n64", Name: "Nintendo 64", Short: "N64", Maker: "Nintendo",
		Year: 1996, Media: "cartridge", Exts: []string{".z64", ".n64", ".v64"},
		Emulator: RetroArch, Core: "mupen64plus_next",
		Palette: Palette{Bg: "#10131e", Surface: "#1e2334", Accent: "#ffcd38", Accent2: "#4cc76b", Text: text, Muted: "#b5b8c4"},
	},
	{
		Slug: "psx", Name: "PlayStation", Short: "PS1", Maker: "Sony",
		Year: 1994, Media: "disc", Exts: []string{".cue", ".chd", ".pbp", ".m3u"},
		Emulator: RetroArch, Core: "mednafen_psx",
		Palette: Palette{Bg: "#14161b", Surface: "#23262e", Accent: "#78a6ff", Accent2: "#f2c14e", Text: text, Muted: "#b3b5bc"},
	},
	{
		Slug: "ps2", Name: "PlayStation 2", Short: "PS2", Maker: "Sony",
		Year: 2000, Media: "disc", Exts: []string{".iso", ".chd", ".cso"},
		Emulator: PCSX2,
		Palette:  Palette{Bg: "#070b19", Surface: "#121a33", Accent: "#6f9bff", Accent2: "#b6c8ff", Text: text, Muted: "#aeb5c9"},
	},
	{
		Slug: "nds", Name: "Nintendo DS", Short: "DS", Maker: "Nintendo",
		Year: 2004, Media: "card", Exts: []string{".nds"},
		Emulator: RetroArch, Core: "melondsds",
		Palette: Palette{Bg: "#14171a", Surface: "#22272c", Accent: "#5ec4f2", Accent2: "#d4d9df", Text: text, Muted: "#b2b7bc"},
	},
	{
		Slug: "psp", Name: "PlayStation Portable", Short: "PSP", Maker: "Sony",
		Year: 2004, Media: "umd", Exts: []string{".iso", ".cso", ".pbp", ".chd"},
		Emulator: PPSSPP,
		Palette:  Palette{Bg: "#0c0c0e", Surface: "#1c1c21", Accent: "#d0d7e0", Accent2: "#5b8cff", Text: text, Muted: "#b0b0b5"},
	},
	{
		Slug: "arcade", Name: "Arcade", Short: "ARC", Maker: "Vários",
		Year: 1980, Media: "arcade", Exts: []string{".zip"},
		Emulator: RetroArch, Core: "fbneo",
		Palette: Palette{Bg: "#140818", Surface: "#26112c", Accent: "#ff5cc6", Accent2: "#38e0ff", Text: text, Muted: "#c2b0c6"},
	},
}

// Find devolve o console pelo slug (aceita os apelidos de pasta mais comuns).
func Find(slug string) (System, bool) {
	slug = strings.ToLower(slug)
	if alias, ok := aliases[slug]; ok {
		slug = alias
	}
	for _, s := range Systems {
		if s.Slug == slug {
			return s, true
		}
	}
	return System{}, false
}

var aliases = map[string]string{
	"megadrive": "genesis", "md": "genesis", "gb": "gbc", "ps1": "psx",
	"playstation": "psx", "fbneo": "arcade", "mame": "arcade", "ds": "nds",
}

// ByExtension lista os consoles que aceitam a extensão do arquivo.
// Extensões ambíguas (.iso, .chd, .zip) devolvem mais de um ou nenhum certo:
// quem chama decide olhando o conteúdo.
func ByExtension(name string) []System {
	ext := strings.ToLower(filepath.Ext(name))
	var out []System
	for _, s := range Systems {
		for _, e := range s.Exts {
			if e == ext {
				out = append(out, s)
				break
			}
		}
	}
	return out
}
