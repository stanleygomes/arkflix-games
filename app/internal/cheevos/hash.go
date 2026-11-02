package cheevos

import (
	"archive/zip"
	"bufio"
	"bytes"
	"crypto/md5"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/stanleygomes/arkflix-games/app/internal/isoinfo"
)

// maxHash: o rcheevos só lê os primeiros 64 MB de qualquer arquivo.
const maxHash = 64 << 20

// Unsupported diz por que não dá pra identificar o jogo (não é falha de rede
// nem de disco: a interface mostra o motivo e segue).
type Unsupported struct{ Reason string }

func (u Unsupported) Error() string { return u.Reason }

func unsupported(format string, a ...any) error { return Unsupported{fmt.Sprintf(format, a...)} }

// Hash calcula o hash do RetroAchievements do arquivo que o emulador abre,
// seguindo o rc_hash do rcheevos. system é o slug do catálogo.
func Hash(path, system string) (string, error) {
	switch system {
	case "arcade":
		return arcade(path), nil // o RA identifica arcade pelo nome do romset
	case "psx", "ps2", "psp":
		return disc(path, system)
	case "nes", "snes", "genesis", "gbc", "n64", "nds":
	default:
		return "", unsupported("console sem conquistas no app")
	}
	data, err := romData(path)
	if err != nil {
		return "", err
	}
	return cartridge(data, system)
}

func sum(parts ...[]byte) string {
	h := md5.New()
	for _, p := range parts {
		h.Write(p)
	}
	return hex.EncodeToString(h.Sum(nil))
}

func arcade(path string) string {
	name := filepath.Base(path)
	return sum([]byte(strings.TrimSuffix(name, filepath.Ext(name))))
}

// romData lê a ROM; num zip, o primeiro arquivo (é o que o RetroArch carrega).
func romData(path string) ([]byte, error) {
	if !strings.EqualFold(filepath.Ext(path), ".zip") {
		f, err := os.Open(path)
		if err != nil {
			return nil, err
		}
		defer f.Close()
		return io.ReadAll(io.LimitReader(f, maxHash))
	}
	z, err := zip.OpenReader(path)
	if err != nil {
		return nil, fmt.Errorf("zip ilegível: %w", err)
	}
	defer z.Close()
	for _, f := range z.File {
		if f.FileInfo().IsDir() {
			continue
		}
		r, err := f.Open()
		if err != nil {
			return nil, err
		}
		defer r.Close()
		return io.ReadAll(io.LimitReader(r, maxHash))
	}
	return nil, unsupported("zip vazio")
}

// cartridge aplica as regras de cada console (cabeçalhos e ordem de bytes).
func cartridge(data []byte, system string) (string, error) {
	switch system {
	case "nes":
		// cabeçalho iNES/FDS de 16 bytes fica de fora
		if len(data) > 16 && (bytes.HasPrefix(data, []byte("NES\x1a")) || bytes.HasPrefix(data, []byte("FDS\x1a"))) {
			data = data[16:]
		}
	case "snes":
		// cabeçalho de copiadora: 512 bytes sobrando de um múltiplo de 8 KB
		if len(data)%0x2000 == 512 {
			data = data[512:]
		}
	case "n64":
		return n64(data)
	case "nds":
		return nds(data)
	}
	return sum(data), nil
}

// n64 normaliza .v64 (16 bits trocados) e .n64 (little endian) pra .z64.
func n64(data []byte) (string, error) {
	if len(data) == 0 {
		return "", unsupported("ROM de N64 vazia")
	}
	out := append([]byte(nil), data...)
	switch data[0] {
	case 0x80, 0xE8, 0x22: // z64 ou 64DD: já na ordem certa
	case 0x37:
		for i := 0; i+1 < len(out); i += 2 {
			out[i], out[i+1] = out[i+1], out[i]
		}
	case 0x40:
		for i := 0; i+3 < len(out); i += 4 {
			out[i], out[i+1], out[i+2], out[i+3] = out[i+3], out[i+2], out[i+1], out[i]
		}
	default:
		return "", unsupported("não parece uma ROM de N64")
	}
	return sum(out), nil
}

// nds: cabeçalho (0x160), código arm9 e arm7 e 0xA00 de ícone/títulos.
func nds(data []byte) (string, error) {
	if len(data) >= 0x200 && bytes.Equal(data[:4], []byte{0x2E, 0, 0, 0xEA}) && bytes.Equal(data[0xB0:0xB4], []byte{0x44, 0x46, 0x96, 0}) {
		data = data[512:] // cabeçalho do SuperCard
	}
	if len(data) < 0x200 {
		return "", unsupported("ROM de DS curta demais")
	}
	u32 := func(off int) int { return int(binary.LittleEndian.Uint32(data[off:])) }
	arm9, arm9Size, arm7, arm7Size, icon := u32(0x20), u32(0x2C), u32(0x30), u32(0x3C), u32(0x68)
	if arm9Size+arm7Size > 16<<20 {
		return "", unsupported("não parece uma ROM de DS")
	}
	slice := func(off, n int) []byte {
		out := make([]byte, n) // o que faltar no fim vira zero, como no rcheevos
		if off < len(data) {
			copy(out, data[off:])
		}
		return out
	}
	return sum(data[:0x160], slice(arm9, arm9Size), slice(arm7, arm7Size), slice(icon, 0xA00)), nil
}

// disc: PS1/PS2 hasheiam nome + conteúdo do executável do SYSTEM.CNF; PSP, o
// PARAM.SFO e o EBOOT.BIN (.pbp: o arquivo inteiro).
func disc(path, system string) (string, error) {
	ext := strings.ToLower(filepath.Ext(path))
	if system == "psp" && ext == ".pbp" {
		data, err := romData(path)
		if err != nil {
			return "", err
		}
		return sum(data), nil
	}
	if ext == ".cue" {
		bin, err := cueTrack1(path)
		if err != nil {
			return "", err
		}
		path, ext = bin, strings.ToLower(filepath.Ext(bin))
	}
	if ext != ".iso" && ext != ".bin" && ext != ".img" {
		return "", unsupported("formato %s ainda sem conquistas (use .cue/.bin ou .iso)", ext)
	}
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	d, err := isoinfo.OpenDisc(f)
	if err != nil {
		return "", unsupported("imagem de disco não reconhecida")
	}
	if system == "psp" {
		return pspDisc(d)
	}
	return bootExe(d, system)
}

// cueTrack1: o primeiro FILE do .cue (a trilha de dados).
func cueTrack1(cue string) (string, error) {
	f, err := os.Open(cue)
	if err != nil {
		return "", err
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if !strings.HasPrefix(strings.ToUpper(line), "FILE ") {
			continue
		}
		name := strings.TrimSpace(line[5:])
		if i := strings.LastIndexByte(name, ' '); i > 0 {
			name = strings.TrimSpace(name[:i]) // tira o tipo (BINARY)
		}
		name = strings.Trim(name, `"`)
		name = filepath.Base(strings.ReplaceAll(name, `\`, "/"))
		return filepath.Join(filepath.Dir(cue), name), nil
	}
	return "", unsupported(".cue sem trilha")
}

func bootExe(d isoinfo.Disc, system string) (string, error) {
	key, prefix := "BOOT", "cdrom:"
	if system == "ps2" {
		key, prefix = "BOOT2", "cdrom0:"
	}
	name := ""
	if lba, size, err := d.Find("SYSTEM.CNF"); err == nil {
		cnf, err := d.ReadAt(lba, int(min(size, 2047)))
		if err != nil {
			return "", err
		}
		name = bootName(string(cnf), key, prefix)
	}
	if name == "" && system == "psx" {
		name = "PSX.EXE"
	}
	if name == "" {
		return "", unsupported("executável de boot não encontrado no disco")
	}
	lba, size, err := d.Find(strings.ReplaceAll(name, `\`, "/"))
	if err != nil {
		return "", unsupported("executável de boot %s não encontrado", name)
	}
	if system == "psx" {
		// o tamanho vem do cabeçalho PS-X EXE (+2048 do próprio cabeçalho)
		if head, err := d.ReadAt(lba, 32); err == nil && string(head[:7]) == "PS-X EX" {
			size = binary.LittleEndian.Uint32(head[28:]) + 2048
		}
	}
	data, err := d.ReadAt(lba, int(min(size, maxHash)))
	if err != nil {
		return "", err
	}
	return sum([]byte(name), data), nil
}

// bootName acha "BOOT = cdrom:\SLUS_000.67;1" e devolve "SLUS_000.67".
func bootName(cnf, key, prefix string) string {
	for _, line := range strings.Split(cnf, "\n") {
		if !strings.HasPrefix(line, key) {
			continue
		}
		rest := strings.TrimSpace(line[len(key):])
		if !strings.HasPrefix(rest, "=") {
			continue
		}
		rest = strings.TrimSpace(rest[1:])
		rest = strings.TrimPrefix(rest, prefix)
		rest = strings.TrimLeft(rest, `\`)
		if i := strings.IndexAny(rest, " \t\r\n;"); i >= 0 {
			rest = rest[:i]
		}
		return rest
	}
	return ""
}

func pspDisc(d isoinfo.Disc) (string, error) {
	var parts [][]byte
	for _, p := range []string{"PSP_GAME/PARAM.SFO", "PSP_GAME/SYSDIR/EBOOT.BIN"} {
		lba, size, err := d.Find(p)
		if err != nil {
			return "", unsupported("disco de PSP sem %s", p)
		}
		data, err := d.ReadAt(lba, int(min(size, maxHash)))
		if err != nil {
			return "", err
		}
		parts = append(parts, data)
	}
	return sum(parts...), nil
}

// IsUnsupported diz se o erro é "não dá pra identificar" (e não falha de disco).
func IsUnsupported(err error) bool {
	var u Unsupported
	return errors.As(err, &u)
}
