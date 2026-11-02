package cheevos_test

import (
	"bytes"
	"crypto/md5"
	"encoding/binary"
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"

	"github.com/stanleygomes/arkflix-games/app/internal/cheevos"
	"github.com/stanleygomes/arkflix-games/app/internal/fixture"
)

func md5hex(parts ...[]byte) string {
	h := md5.New()
	for _, p := range parts {
		h.Write(p)
	}
	return hex.EncodeToString(h.Sum(nil))
}

func write(t *testing.T, name string, data []byte) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(p, data, 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

func hash(t *testing.T, path, system string) string {
	t.Helper()
	h, err := cheevos.Hash(path, system)
	if err != nil {
		t.Fatalf("hash de %s: %v", filepath.Base(path), err)
	}
	return h
}

func seq(n int) []byte {
	b := make([]byte, n)
	for i := range b {
		b[i] = byte(i*7 + 3)
	}
	return b
}

func TestNESHeader(t *testing.T) {
	body := seq(1024)
	header := append([]byte("NES\x1a"), make([]byte, 12)...)
	want := md5hex(body)
	if h := hash(t, write(t, "a.nes", append(header, body...)), "nes"); h != want {
		t.Fatal("cabeçalho iNES devia ficar de fora")
	}
	if h := hash(t, write(t, "b.nes", body), "nes"); h != want {
		t.Fatal("sem cabeçalho: o arquivo inteiro")
	}
	zipped := fixture.ZipBytes(map[string]string{"smb.nes": string(append(header, body...))})
	if h := hash(t, write(t, "c.zip", zipped), "nes"); h != want {
		t.Fatal("dentro do zip: mesma regra")
	}
}

func TestSNESCopierHeader(t *testing.T) {
	body := seq(0x2000)
	want := md5hex(body)
	if h := hash(t, write(t, "a.smc", append(make([]byte, 512), body...)), "snes"); h != want {
		t.Fatal("cabeçalho de copiadora devia sair")
	}
	if h := hash(t, write(t, "b.sfc", body), "snes"); h != want {
		t.Fatal("sem cabeçalho: inteiro")
	}
	odd := seq(0x2000 + 100)
	if h := hash(t, write(t, "c.sfc", odd), "snes"); h != md5hex(odd) {
		t.Fatal("sobra que não é 512 não é cabeçalho")
	}
}

func TestN64ByteOrder(t *testing.T) {
	z64 := append([]byte{0x80, 0x37, 0x12, 0x40}, seq(60)...)
	v64 := append([]byte(nil), z64...)
	for i := 0; i < len(v64); i += 2 {
		v64[i], v64[i+1] = v64[i+1], v64[i]
	}
	n64 := append([]byte(nil), z64...)
	for i := 0; i < len(n64); i += 4 {
		n64[i], n64[i+1], n64[i+2], n64[i+3] = n64[i+3], n64[i+2], n64[i+1], n64[i]
	}
	want := md5hex(z64)
	for name, data := range map[string][]byte{"a.z64": z64, "a.v64": v64, "a.n64": n64} {
		if h := hash(t, write(t, name, data), "n64"); h != want {
			t.Fatalf("%s devia virar z64", name)
		}
	}
	if _, err := cheevos.Hash(write(t, "x.z64", []byte("N64")), "n64"); !cheevos.IsUnsupported(err) {
		t.Fatalf("lixo não é ROM de N64: %v", err)
	}
}

func TestZipFirstEntry(t *testing.T) {
	// o zip guarda na ordem de criação; o primeiro é o que o RetroArch carrega
	var buf bytes.Buffer
	zw := newZip(&buf, []string{"game.md", "readme.txt"}, [][]byte{[]byte("MEGA"), []byte("leia")})
	zw.Close()
	if h := hash(t, write(t, "g.zip", buf.Bytes()), "genesis"); h != md5hex([]byte("MEGA")) {
		t.Fatal("devia hashear o primeiro arquivo do zip")
	}
}

func TestArcadeByName(t *testing.T) {
	p := write(t, "mspacman.zip", fixture.ZipBytes(map[string]string{"pacman.6e": "x"}))
	if h := hash(t, p, "arcade"); h != md5hex([]byte("mspacman")) {
		t.Fatal("arcade é pelo nome do romset")
	}
}

func psxExe(size int) []byte {
	exe := seq(size)
	copy(exe, "PS-X EXE")
	binary.LittleEndian.PutUint32(exe[28:], uint32(size-2048))
	return exe
}

func TestPS1(t *testing.T) {
	exe := psxExe(4096)
	iso := fixture.BuildISO(map[string][]byte{
		"SYSTEM.CNF":  []byte("BOOT = cdrom:\\SLUS_000.67;1\r\nTCB = 4\r\n"),
		"SLUS_000.67": append(exe, seq(3000)...), // sobra além do tamanho do cabeçalho
	})
	want := md5hex([]byte("SLUS_000.67"), exe)
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "Jogo (USA).bin"), fixture.RawBin(iso, true), 0o644)
	cue := filepath.Join(dir, "Jogo (USA).cue")
	os.WriteFile(cue, []byte("FILE \"Jogo (USA).bin\" BINARY\n  TRACK 01 MODE2/2352\n"), 0o644)
	if h := hash(t, cue, "psx"); h != want {
		t.Fatal("PS1 .cue/.bin: nome + executável do SYSTEM.CNF")
	}
	if _, err := cheevos.Hash(write(t, "x.chd", []byte("x")), "psx"); !cheevos.IsUnsupported(err) {
		t.Fatalf(".chd ainda não: %v", err)
	}
}

func TestPS2(t *testing.T) {
	elf := append([]byte("\x7fELF"), seq(5000)...)
	iso := fixture.BuildISO(map[string][]byte{
		"SYSTEM.CNF":  []byte("BOOT2 = cdrom0:\\SLUS_214.31;1\nVER = 1.00\n"),
		"SLUS_214.31": elf,
	})
	if h := hash(t, write(t, "fifa.iso", iso), "ps2"); h != md5hex([]byte("SLUS_214.31"), elf) {
		t.Fatal("PS2: nome + ELF inteiro")
	}
}

func TestPSP(t *testing.T) {
	sfo := fixture.SFO(map[string]string{"DISC_ID": "ULUS10084"})
	eboot := seq(3000)
	iso := fixture.BuildISO(map[string][]byte{"PSP_GAME/PARAM.SFO": sfo, "PSP_GAME/SYSDIR/EBOOT.BIN": eboot})
	if h := hash(t, write(t, "tr.iso", iso), "psp"); h != md5hex(sfo, eboot) {
		t.Fatal("PSP: PARAM.SFO + EBOOT.BIN")
	}
	if _, err := cheevos.Hash(write(t, "x.iso", fixture.PSPISO("ULUS10084", "x")), "psp"); !cheevos.IsUnsupported(err) {
		t.Fatalf("sem EBOOT.BIN: %v", err)
	}
}

func TestFixtureGameHash(t *testing.T) {
	env := fixture.NewEnv(t.TempDir())
	p := filepath.Join(env.Local, "snes", "Super Mario World (U) [!].zip")
	if h := hash(t, p, "snes"); h != fixture.FakeRAHash {
		t.Fatalf("o RA falso devia conhecer o SMW do ambiente falso: %s", h)
	}
}
