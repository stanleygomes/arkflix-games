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

const text = "#e8fbff" // branco frio de tubo de TV

// Systems na ordem em que aparecem na barra de consoles.
var Systems = []System{
	{
		Slug: "nes", Name: "Nintendo Entertainment System", Short: "NES", Maker: "Nintendo",
		Year: 1983, Media: "cartridge", Exts: []string{".nes", ".unf", ".fds"},
		Emulator: RetroArch, Core: "mesen",
		Palette: Palette{Bg: "#07050a", Surface: "#140c18", Accent: "#ff4d6a", Accent2: "#18f0ff", Text: text, Muted: "#bfb4c6"},
	},
	{
		Slug: "snes", Name: "Super Nintendo", Short: "SNES", Maker: "Nintendo",
		Year: 1990, Media: "cartridge", Exts: []string{".sfc", ".smc", ".swc", ".fig"},
		Emulator: RetroArch, Core: "snes9x",
		Palette: Palette{Bg: "#08050f", Surface: "#160d26", Accent: "#b896ff", Accent2: "#ffe600", Text: text, Muted: "#c0b6d2"},
	},
	{
		Slug: "genesis", Name: "Mega Drive", Short: "MD", Maker: "Sega",
		Year: 1988, Media: "cartridge", Exts: []string{".md", ".gen", ".smd", ".bin", ".sms", ".gg"},
		Emulator: RetroArch, Core: "genesis_plus_gx",
		Palette: Palette{Bg: "#04060d", Surface: "#0c1224", Accent: "#4d94ff", Accent2: "#ff4d6a", Text: text, Muted: "#b4bccf"},
	},
	{
		Slug: "gbc", Name: "Game Boy Color", Short: "GBC", Maker: "Nintendo",
		Year: 1998, Media: "cartridge", Exts: []string{".gb", ".gbc", ".sgb"},
		Emulator: RetroArch, Core: "gambatte",
		Palette: Palette{Bg: "#030b05", Surface: "#0b1c10", Accent: "#7dff4f", Accent2: "#ffe600", Text: text, Muted: "#b4c8b4"},
	},
	{
		Slug: "n64", Name: "Nintendo 64", Short: "N64", Maker: "Nintendo",
		Year: 1996, Media: "cartridge", Exts: []string{".z64", ".n64", ".v64"},
		Emulator: RetroArch, Core: "mupen64plus_next",
		Palette: Palette{Bg: "#050608", Surface: "#10131c", Accent: "#ffe600", Accent2: "#3dff8b", Text: text, Muted: "#bcbfc8"},
	},
	{
		Slug: "psx", Name: "PlayStation", Short: "PS1", Maker: "Sony",
		Year: 1994, Media: "disc", Exts: []string{".cue", ".chd", ".pbp", ".m3u"},
		Emulator: RetroArch, Core: "mednafen_psx",
		Palette: Palette{Bg: "#05060a", Surface: "#10121c", Accent: "#18f0ff", Accent2: "#ff3cac", Text: text, Muted: "#b8bbc8"},
	},
	{
		Slug: "ps2", Name: "PlayStation 2", Short: "PS2", Maker: "Sony",
		Year: 2000, Media: "disc", Exts: []string{".iso", ".chd", ".cso"},
		Emulator: PCSX2,
		Palette:  Palette{Bg: "#02040e", Surface: "#0a122e", Accent: "#5c8cff", Accent2: "#18f0ff", Text: text, Muted: "#b0b8d0"},
	},
	{
		Slug: "nds", Name: "Nintendo DS", Short: "DS", Maker: "Nintendo",
		Year: 2004, Media: "card", Exts: []string{".nds"},
		Emulator: RetroArch, Core: "melondsds",
		Palette: Palette{Bg: "#05080a", Surface: "#0e161b", Accent: "#5ee6ff", Accent2: "#ff3cac", Text: text, Muted: "#b4bec4"},
	},
	{
		Slug: "psp", Name: "PlayStation Portable", Short: "PSP", Maker: "Sony",
		Year: 2004, Media: "umd", Exts: []string{".iso", ".cso", ".pbp", ".chd"},
		Emulator: PPSSPP,
		Palette:  Palette{Bg: "#050505", Surface: "#121214", Accent: "#e8f0ff", Accent2: "#18f0ff", Text: text, Muted: "#b8b8bf"},
	},
	{
		Slug: "arcade", Name: "Arcade", Short: "ARC", Maker: "Vários",
		Year: 1980, Media: "arcade", Exts: []string{".zip"},
		Emulator: RetroArch, Core: "fbneo",
		Palette: Palette{Bg: "#050008", Surface: "#13061c", Accent: "#ff3cac", Accent2: "#18f0ff", Text: text, Muted: "#c0b0cc"},
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
