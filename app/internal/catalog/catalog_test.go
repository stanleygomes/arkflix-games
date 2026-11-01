package catalog_test

import (
	"math"
	"strconv"
	"testing"

	"github.com/stanleygomes/arkflix-games/app/internal/catalog"
)

func lum(hex string) float64 {
	n, _ := strconv.ParseUint(hex[1:], 16, 32)
	ch := func(v uint64) float64 {
		s := float64(v) / 255
		if s <= 0.03928 {
			return s / 12.92
		}
		return math.Pow((s+0.055)/1.055, 2.4)
	}
	return 0.2126*ch(n>>16&255) + 0.7152*ch(n>>8&255) + 0.0722*ch(n&255)
}

func contrast(a, b string) float64 {
	la, lb := lum(a), lum(b)
	if la < lb {
		la, lb = lb, la
	}
	return (la + 0.05) / (lb + 0.05)
}

// Acessibilidade: texto e texto secundário com contraste AA (4.5:1) sobre o
// fundo e a superfície; cor de destaque (foco, bordas) com 3:1 (componente de UI).
func TestPalettesContrast(t *testing.T) {
	for _, s := range catalog.Systems {
		p := s.Palette
		checks := []struct {
			name   string
			fg, bg string
			min    float64
		}{
			{"texto/fundo", p.Text, p.Bg, 4.5},
			{"texto/superfície", p.Text, p.Surface, 4.5},
			{"secundário/fundo", p.Muted, p.Bg, 4.5},
			{"secundário/superfície", p.Muted, p.Surface, 4.5},
			{"destaque/fundo", p.Accent, p.Bg, 3},
			{"destaque2/fundo", p.Accent2, p.Bg, 3},
		}
		for _, c := range checks {
			if got := contrast(c.fg, c.bg); got < c.min {
				t.Errorf("%s %s: contraste %.2f < %.1f (%s em %s)", s.Slug, c.name, got, c.min, c.fg, c.bg)
			}
		}
	}
}

func TestCatalog(t *testing.T) {
	seen := map[string]bool{}
	for _, s := range catalog.Systems {
		if seen[s.Slug] {
			t.Errorf("slug repetido: %s", s.Slug)
		}
		seen[s.Slug] = true
		switch s.Emulator {
		case catalog.RetroArch:
			if s.Core == "" {
				t.Errorf("%s usa RetroArch sem core", s.Slug)
			}
		case catalog.PPSSPP, catalog.PCSX2:
		default:
			t.Errorf("%s: emulador desconhecido %q", s.Slug, s.Emulator)
		}
		if len(s.Exts) == 0 || s.Short == "" || s.Name == "" {
			t.Errorf("%s incompleto", s.Slug)
		}
	}
}

func TestFindAliases(t *testing.T) {
	for alias, want := range map[string]string{"megadrive": "genesis", "PS1": "psx", "gb": "gbc", "SNES": "snes"} {
		if s, ok := catalog.Find(alias); !ok || s.Slug != want {
			t.Errorf("Find(%q) = %v %v, esperado %s", alias, s.Slug, ok, want)
		}
	}
	if _, ok := catalog.Find("dreamcast"); ok {
		t.Error("dreamcast não é suportado")
	}
}

func TestByExtension(t *testing.T) {
	if got := catalog.ByExtension("jogo.SFC"); len(got) != 1 || got[0].Slug != "snes" {
		t.Errorf(".sfc: %v", got)
	}
	if got := catalog.ByExtension("jogo.iso"); len(got) != 2 {
		t.Errorf(".iso devia ser ambíguo (psp, ps2): %v", got)
	}
}
