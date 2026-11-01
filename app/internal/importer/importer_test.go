package importer_test

import (
	"archive/zip"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stanleygomes/arkflix-games/app/internal/fixture"
	"github.com/stanleygomes/arkflix-games/app/internal/importer"
)

func write(t *testing.T, path string, data []byte) string {
	t.Helper()
	os.MkdirAll(filepath.Dir(path), 0o755)
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func zipOf(t *testing.T, path string, files map[string]string) string {
	t.Helper()
	os.MkdirAll(filepath.Dir(path), 0o755)
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	w := zip.NewWriter(f)
	for name, body := range files {
		fw, _ := w.Create(name)
		fw.Write([]byte(body))
	}
	w.Close()
	f.Close()
	return path
}

func TestAnalyze(t *testing.T) {
	src := t.TempDir()
	paths := map[string]string{
		"snes":    write(t, filepath.Join(src, "Super Mario World (U).sfc"), []byte("rom")),
		"zipnes":  zipOf(t, filepath.Join(src, "SMB3.zip"), map[string]string{"SMB3.nes": "rom"}),
		"arcade":  zipOf(t, filepath.Join(src, "mspacman.zip"), map[string]string{"pacman.6e": "x", "pacman.6f": "y"}),
		"psxzip":  zipOf(t, filepath.Join(src, "Crash (USA).zip"), map[string]string{"Crash (USA).cue": "FILE \"Crash (USA).bin\" BINARY", "Crash (USA).bin": "b"}),
		"psp":     write(t, filepath.Join(src, "lara.iso"), fixture.PSPISO("ULUS10041", "Lara Croft Legend")),
		"ps2":     write(t, filepath.Join(src, "fifa.iso"), fixture.PS2ISO("SLUS-21431")),
		"gdi":     write(t, filepath.Join(src, "crazy.gdi"), []byte("x")),
		"unknown": write(t, filepath.Join(src, "coisa.xyz"), []byte("x")),
		"chd":     write(t, filepath.Join(src, "jogo.chd"), []byte("MComprHD")),
	}
	items := importer.Analyze([]string{
		paths["snes"], paths["zipnes"], paths["arcade"], paths["psxzip"], paths["psp"],
		paths["ps2"], paths["gdi"], paths["unknown"], paths["chd"], filepath.Join(src, "sumiu.nes"),
	})
	want := []struct {
		system, problem string
		extract         bool
	}{
		{"snes", "", false}, {"nes", "", false}, {"arcade", "", false}, {"psx", "", true},
		{"psp", "", false}, {"ps2", "", false}, {"", "Dreamcast", false},
		{"", "extensão desconhecida", false}, {"", "", false}, {"", "não encontrado", false},
	}
	for i, w := range want {
		it := items[i]
		if it.System != w.system || it.Extract != w.extract || !strings.Contains(it.Problem, w.problem) {
			t.Errorf("%s: got system=%q extract=%v problem=%q", filepath.Base(it.Source), it.System, it.Extract, it.Problem)
		}
	}
	if items[4].Title != "Lara Croft Legend" {
		t.Errorf("título da ISO de PSP devia vir do PARAM.SFO: %q", items[4].Title)
	}
	if len(items[8].Candidates) < 3 {
		t.Errorf(".chd é ambíguo, devia oferecer vários consoles: %v", items[8].Candidates)
	}
}

func TestImport(t *testing.T) {
	src, lib := t.TempDir(), t.TempDir()
	rom := write(t, filepath.Join(src, "Tetris DX.gbc"), []byte("rom"))
	disc := zipOf(t, filepath.Join(src, "Crash (USA).zip"), map[string]string{
		"Crash/Crash (USA).cue": "FILE \"C:\\jogos\\Crash (USA).bin\" BINARY\r\n  TRACK 01 MODE2/2352\r\n",
		"Crash/Crash (USA).bin": "bin",
	})
	cue := write(t, filepath.Join(src, "solto", "Spyro.cue"), []byte("FILE \"Spyro.bin\" BINARY\n"))
	write(t, filepath.Join(src, "solto", "Spyro.bin"), []byte("b"))

	items := importer.Analyze([]string{rom, disc, cue})
	results := importer.Import(items, lib)
	for _, r := range results {
		if !r.OK {
			t.Fatalf("%s falhou: %s", r.Source, r.Problem)
		}
	}
	if _, err := os.Stat(filepath.Join(lib, "gbc", "Tetris DX.gbc")); err != nil {
		t.Error("gbc não copiado")
	}
	cueData, err := os.ReadFile(filepath.Join(lib, "psx", "Crash", "Crash (USA).cue"))
	if err != nil {
		t.Fatalf("disco não extraído em pasta: %v", err)
	}
	if !strings.HasPrefix(string(cueData), `FILE "Crash (USA).bin" BINARY`) || !strings.Contains(string(cueData), "\r\n") {
		t.Errorf("FILE do cue não corrigido: %q", cueData)
	}
	if _, err := os.Stat(filepath.Join(lib, "psx", "Spyro", "Spyro.bin")); err != nil {
		t.Error(".cue solto devia levar o .bin junto pra uma pasta")
	}

	again := importer.Import(importer.Analyze([]string{rom}), lib)
	if again[0].OK || !strings.Contains(again[0].Problem, "já existe") {
		t.Errorf("reimportar devia recusar: %+v", again[0])
	}
	leftovers, _ := filepath.Glob(filepath.Join(lib, "*", ".import-*"))
	if len(leftovers) > 0 {
		t.Errorf("sobrou pasta temporária: %v", leftovers)
	}
}

func TestImportNeedsSystem(t *testing.T) {
	lib := t.TempDir()
	path := write(t, filepath.Join(t.TempDir(), "jogo.chd"), []byte("x"))
	items := importer.Analyze([]string{path})
	if r := importer.Import(items, lib); r[0].OK {
		t.Fatal("sem console escolhido não podia importar")
	}
	items[0].System = "psx"
	if r := importer.Import(items, lib); !r[0].OK {
		t.Fatalf("com console escolhido devia importar: %s", r[0].Problem)
	}
}

func TestCueMissingBin(t *testing.T) {
	src, lib := t.TempDir(), t.TempDir()
	cue := write(t, filepath.Join(src, "Sem Bin.cue"), []byte("FILE \"Sem Bin.bin\" BINARY\n"))
	r := importer.Import(importer.Analyze([]string{cue}), lib)
	if r[0].OK || !strings.Contains(r[0].Problem, "não está junto") {
		t.Fatalf("cue sem bin devia falhar com motivo claro: %+v", r[0])
	}
}
