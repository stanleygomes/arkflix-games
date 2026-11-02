// Package bios sabe que BIOS cada console precisa (nome, tamanho e MD5
// conhecidos, das docs do libretro e do PCSX2) e confere se ela está na pasta
// do emulador. O MD5 e a assinatura do PS2 ficam em memória por arquivo
// (caminho + tamanho + data): conferir de novo só lê o que mudou.
package bios

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// Onde a BIOS fica.
const (
	RetroArchSystem = "retroarch" // pasta system do RetroArch
	PCSX2Bios       = "pcsx2"     // pasta bios do PCSX2
)

// File é um arquivo de BIOS conhecido.
type File struct {
	Name string `json:"file"`
	MD5  string `json:"md5,omitempty"`
	Note string `json:"note,omitempty"`
	Size int64  `json:"-"`
}

// Mode: como os arquivos da regra contam.
type Mode int

const (
	Any  Mode = iota // basta um dos arquivos
	Each             // cada arquivo vale por si (opcionais)
	PS2              // qualquer BIOS de PS2 na pasta (pela assinatura)
)

// Rule é o que um console precisa.
type Rule struct {
	System   string
	Required bool // sem ela o emulador não abre o jogo
	Where    string
	Mode     Mode
	Files    []File
	Hint     string // como dizer o que falta na mensagem do Play
}

const (
	ps1Size  = 512 << 10
	ps2Min   = 4 << 20
	ps2Max   = 8 << 20
	ps2Label = "BIOS do PS2 (.bin de 4 MB)"
)

// Rules: só os consoles que usam BIOS. PSP, N64, NES, SNES, Mega Drive e Game
// Boy não precisam de nada.
var Rules = []Rule{
	{
		System: "psx", Required: true, Where: RetroArchSystem, Mode: Any,
		Hint: "scph5501.bin etc.",
		Files: []File{
			{Name: "scph5501.bin", MD5: "490f666e1afb15b7362b406ed1cea246", Size: ps1Size, Note: "EUA"},
			{Name: "scph5500.bin", MD5: "8dd7d5296a650fac7319bce665a6a53c", Size: ps1Size, Note: "Japão"},
			{Name: "scph5502.bin", MD5: "32736f17079d0b2b7024407c39bd3050", Size: ps1Size, Note: "Europa"},
		},
	},
	{
		System: "ps2", Required: true, Where: PCSX2Bios, Mode: PS2,
		Hint:  "arquivo .bin de 4 MB do seu console",
		Files: []File{{Name: ps2Label, Note: "qualquer BIOS de PS2, tirada do seu console"}},
	},
	{
		System: "nds", Where: RetroArchSystem, Mode: Each,
		Files: []File{
			{Name: "bios7.bin", MD5: "df692a80a5b1bc90728bc3dfc76cd948", Size: 16 << 10, Note: "opcional"},
			{Name: "bios9.bin", MD5: "a392174eb3e572fed6447e956bde4b25", Size: 4 << 10, Note: "opcional"},
			{Name: "firmware.bin", Note: "opcional"},
		},
	},
	{
		System: "arcade", Where: RetroArchSystem, Mode: Each,
		Files: []File{{Name: "neogeo.zip", Note: "só pros jogos de Neo Geo"}},
	},
}

// Find devolve a regra do console (falso = não precisa de BIOS).
func Find(system string) (Rule, bool) {
	for _, r := range Rules {
		if r.System == system {
			return r, true
		}
	}
	return Rule{}, false
}

// Result é o que a conferência achou numa pasta.
type Result struct {
	OK       bool
	Found    []string
	Missing  []File
	Warnings []string
}

type key struct {
	path string
	size int64
	mod  time.Time
}

// Checker confere as pastas e lembra o que já leu.
type Checker struct {
	mu   sync.Mutex
	memo map[key]string
}

// NewChecker cria um conferidor com memória vazia.
func NewChecker() *Checker { return &Checker{memo: map[key]string{}} }

// memoized roda fn uma vez por versão do arquivo.
func (c *Checker) memoized(path string, st os.FileInfo, kind string, fn func() string) string {
	k := key{path: kind + "\x00" + path, size: st.Size(), mod: st.ModTime()}
	c.mu.Lock()
	v, ok := c.memo[k]
	c.mu.Unlock()
	if ok {
		return v
	}
	v = fn()
	c.mu.Lock()
	c.memo[k] = v
	c.mu.Unlock()
	return v
}

func fileMD5(path string) string {
	f, err := os.Open(path)
	if err != nil {
		return ""
	}
	defer f.Close()
	h := md5.New()
	if _, err := io.Copy(h, f); err != nil {
		return ""
	}
	return hex.EncodeToString(h.Sum(nil))
}

// IsPS2 diz se o arquivo parece uma BIOS de PS2, como o PCSX2 confere: de 4 a
// 8 MB e com a entrada "RESET" do ROMDIR (tabela de 16 bytes por entrada) no
// começo do arquivo.
func IsPS2(path string) bool {
	st, err := os.Stat(path)
	if err != nil || !st.Mode().IsRegular() || st.Size() < ps2Min || st.Size() > ps2Max {
		return false
	}
	f, err := os.Open(path)
	if err != nil {
		return false
	}
	defer f.Close()
	buf := make([]byte, 512<<10)
	n, _ := io.ReadFull(f, buf)
	sig := []byte("RESET\x00")
	for off := 0; off+16 <= n; off += 16 {
		if bytes.Equal(buf[off:off+len(sig)], sig) {
			return true
		}
	}
	return false
}

// Check confere a regra na pasta dir.
func (c *Checker) Check(r Rule, dir string) Result {
	var res Result
	if r.Mode == PS2 {
		entries, _ := os.ReadDir(dir)
		for _, e := range entries {
			path := filepath.Join(dir, e.Name())
			st, err := os.Stat(path)
			if err != nil || !st.Mode().IsRegular() || st.Size() < ps2Min || st.Size() > ps2Max {
				continue
			}
			if c.memoized(path, st, "ps2", func() string {
				if IsPS2(path) {
					return "1"
				}
				return ""
			}) != "" {
				res.Found = append(res.Found, e.Name())
			}
		}
		res.OK = len(res.Found) > 0
		if !res.OK {
			res.Missing = r.Files
		}
		return res
	}
	for _, f := range r.Files {
		path := filepath.Join(dir, f.Name)
		st, err := os.Stat(path)
		if err != nil || !st.Mode().IsRegular() || (f.Size > 0 && st.Size() != f.Size) {
			if err == nil && st.Mode().IsRegular() {
				res.Warnings = append(res.Warnings, f.Name+": tamanho errado, não serve")
			}
			res.Missing = append(res.Missing, f)
			continue
		}
		res.Found = append(res.Found, f.Name)
		if f.MD5 != "" {
			sum := c.memoized(path, st, "md5", func() string { return fileMD5(path) })
			if !strings.EqualFold(sum, f.MD5) {
				res.Warnings = append(res.Warnings, f.Name+": versão diferente da conhecida (pode não funcionar)")
			}
		}
	}
	switch r.Mode {
	case Any:
		res.OK = len(res.Found) > 0
		if res.OK {
			res.Missing = nil // os outros só fazem falta pra outras regiões
		}
	case Each:
		res.OK = len(res.Missing) == 0
	}
	return res
}
