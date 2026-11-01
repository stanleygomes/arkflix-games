package archive_test

import (
	"archive/zip"
	"os"
	"path/filepath"
	"testing"

	"github.com/stanleygomes/arkflix-games/app/internal/archive"
)

func makeZip(t *testing.T, files map[string]string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "pack.zip")
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	w := zip.NewWriter(f)
	for name, body := range files {
		fw, err := w.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		fw.Write([]byte(body))
	}
	w.Close()
	f.Close()
	return path
}

func TestDetectAndList(t *testing.T) {
	path := makeZip(t, map[string]string{"Game (USA).sfc": "rom", "docs/readme.txt": "oi"})
	kind, err := archive.Detect(path)
	if err != nil || kind != archive.Zip {
		t.Fatalf("detect: %v %v", kind, err)
	}
	entries, err := archive.List(path)
	if err != nil || len(entries) != 2 {
		t.Fatalf("list: %+v %v", entries, err)
	}
}

func TestNotArchive(t *testing.T) {
	path := filepath.Join(t.TempDir(), "rom.sfc")
	os.WriteFile(path, []byte("SNES ROM DATA"), 0o644)
	kind, _ := archive.Detect(path)
	if kind != archive.None {
		t.Fatalf("rom solta detectada como %q", kind)
	}
	if _, err := archive.List(path); err == nil {
		t.Fatal("List devia falhar em arquivo que não é pacote")
	}
}

func TestCorruptZip(t *testing.T) {
	path := makeZip(t, map[string]string{"a.nes": "conteudo grande o bastante"})
	data, _ := os.ReadFile(path)
	os.WriteFile(path, data[:len(data)/2], 0o644)
	if _, err := archive.List(path); err == nil {
		t.Fatal("zip cortado ao meio devia falhar")
	}
}

func TestExtract(t *testing.T) {
	path := makeZip(t, map[string]string{"disc/Game.cue": "FILE", "disc/Game.bin": "BIN"})
	dest := t.TempDir()
	if err := archive.Extract(path, dest); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(filepath.Join(dest, "disc", "Game.bin"))
	if err != nil || string(got) != "BIN" {
		t.Fatalf("extraído errado: %q %v", got, err)
	}
}

func TestExtractRejectsZipSlip(t *testing.T) {
	path := makeZip(t, map[string]string{"../../fora.txt": "mal"})
	dest := filepath.Join(t.TempDir(), "dest")
	if err := archive.Extract(path, dest); err == nil {
		t.Fatal("caminho com .. devia ser recusado")
	}
	if _, err := os.Stat(filepath.Join(filepath.Dir(dest), "fora.txt")); err == nil {
		t.Fatal("arquivo escapou da pasta de destino")
	}
}
