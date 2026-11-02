package bios

import (
	"os"
	"path/filepath"
	"testing"
)

func write(t *testing.T, path string, data []byte) {
	t.Helper()
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestPS1ByNameSizeAndMD5(t *testing.T) {
	dir := t.TempDir()
	c := NewChecker()
	r, _ := Find("psx")
	if res := c.Check(r, dir); res.OK || len(res.Missing) != 3 {
		t.Fatalf("pasta vazia: %+v", res)
	}
	write(t, filepath.Join(dir, "scph5500.bin"), []byte("pequeno demais"))
	if res := c.Check(r, dir); res.OK || len(res.Warnings) != 1 {
		t.Fatalf("tamanho errado não serve: %+v", res)
	}
	write(t, filepath.Join(dir, "scph5501.bin"), make([]byte, 512<<10))
	res := c.Check(r, dir)
	if !res.OK || len(res.Found) != 1 || res.Found[0] != "scph5501.bin" || len(res.Missing) != 0 {
		t.Fatalf("com a BIOS dos EUA: %+v", res)
	}
	// zeros não batem com o MD5 conhecido: serve, mas avisa
	if len(res.Warnings) != 2 {
		t.Fatalf("avisos: %v", res.Warnings)
	}
}

func TestPS2BySignature(t *testing.T) {
	dir := t.TempDir()
	c := NewChecker()
	r, _ := Find("ps2")
	write(t, filepath.Join(dir, "lixo.bin"), make([]byte, 4<<20)) // tamanho certo, sem ROMDIR
	write(t, filepath.Join(dir, "pequeno.bin"), []byte("RESET\x00"))
	if res := c.Check(r, dir); res.OK || len(res.Missing) != 1 {
		t.Fatalf("sem BIOS de verdade: %+v", res)
	}
	b := make([]byte, 4<<20)
	copy(b[0x2700:], "RESET")
	write(t, filepath.Join(dir, "SCPH-70012.BIN"), b)
	if res := c.Check(r, dir); !res.OK || len(res.Found) != 1 || res.Found[0] != "SCPH-70012.BIN" {
		t.Fatalf("com BIOS: %+v", res)
	}
}

func TestOptionalAndNoBios(t *testing.T) {
	for _, slug := range []string{"psp", "n64", "nes", "snes", "genesis", "gbc"} {
		if _, ok := Find(slug); ok {
			t.Errorf("%s não precisa de BIOS", slug)
		}
	}
	r, _ := Find("nds")
	if r.Required {
		t.Fatal("BIOS do DS é opcional")
	}
	if res := NewChecker().Check(r, t.TempDir()); res.OK || len(res.Missing) != 3 {
		t.Fatalf("DS sem nada: %+v", res)
	}
}
