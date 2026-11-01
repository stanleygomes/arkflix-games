// Package isoinfo lê o mínimo de uma imagem de disco ISO 9660 pra identificar o
// jogo: PSP (PSP_GAME/PARAM.SFO: DISC_ID, título), PS2 e PS1 (SYSTEM.CNF: serial).
// Aceita .iso (setor de 2048 bytes) e .bin cru de CD (2352 bytes, modo 1 ou 2).
// Serve pra achar os save states de cada jogo e pra detectar o console ao importar.
package isoinfo

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"
)

const sector = 2048

// ErrNotFound indica que o arquivo procurado não existe na imagem.
var ErrNotFound = errors.New("arquivo não encontrado na ISO")

// Info é o que dá pra saber do jogo.
type Info struct {
	System  string `json:"system"` // "psp" ou "ps2"
	Serial  string `json:"serial"` // PSP: ULUS10041; PS2: SLUS-21431
	Version string `json:"version,omitempty"`
	Title   string `json:"title,omitempty"`
}

type record struct {
	extent uint32
	size   uint32
	dir    bool
	name   string
}

// image lê setores lógicos de 2048 bytes de um .iso ou de um .bin cru.
type image struct {
	r      io.ReaderAt
	raw    int64 // tamanho do setor no arquivo (2048 ou 2352)
	offset int64 // onde começam os dados dentro do setor cru
}

// open descobre o formato procurando o descritor "CD001" no setor 16.
func open(r io.ReaderAt) (image, error) {
	head := make([]byte, 6)
	for _, f := range []struct{ raw, offset int64 }{{sector, 0}, {2352, 24}, {2352, 16}} {
		if _, err := r.ReadAt(head, 16*f.raw+f.offset); err == nil && head[0] == 1 && string(head[1:6]) == "CD001" {
			return image{r: r, raw: f.raw, offset: f.offset}, nil
		}
	}
	return image{}, errors.New("não é uma imagem de disco ISO 9660")
}

// readAt lê n bytes a partir do setor lógico, pulando os cabeçalhos do setor cru.
func (im image) readAt(lba uint32, n int) ([]byte, error) {
	out := make([]byte, 0, n)
	buf := make([]byte, sector)
	for s := int64(lba); len(out) < n; s++ {
		if _, err := im.r.ReadAt(buf, s*im.raw+im.offset); err != nil && !(errors.Is(err, io.EOF) && len(out)+sector >= n) {
			return nil, err
		}
		out = append(out, buf[:min(sector, n-len(out))]...)
	}
	return out, nil
}

func (im image) root() (record, error) {
	pvd, err := im.readAt(16, sector)
	if err != nil {
		return record{}, fmt.Errorf("sem descritor ISO 9660: %w", err)
	}
	return parseRecord(pvd[156:190])
}

func parseRecord(b []byte) (record, error) {
	if len(b) < 34 || int(b[0]) > len(b) {
		return record{}, errors.New("registro de diretório inválido")
	}
	nameLen := int(b[32])
	if 33+nameLen > len(b) {
		return record{}, errors.New("nome de arquivo inválido")
	}
	name := string(b[33 : 33+nameLen])
	if i := strings.IndexByte(name, ';'); i >= 0 {
		name = name[:i]
	}
	return record{
		extent: binary.LittleEndian.Uint32(b[2:6]),
		size:   binary.LittleEndian.Uint32(b[10:14]),
		dir:    b[25]&2 != 0,
		name:   name,
	}, nil
}

func (im image) list(dir record) ([]record, error) {
	if dir.size > 4<<20 {
		return nil, errors.New("diretório grande demais")
	}
	data, err := im.readAt(dir.extent, int(dir.size))
	if err != nil {
		return nil, err
	}
	var out []record
	for off := 0; off < len(data); {
		n := int(data[off])
		if n == 0 { // registros não cruzam setor: pula pro próximo
			off = (off/sector + 1) * sector
			continue
		}
		if off+n > len(data) {
			break
		}
		rec, err := parseRecord(data[off : off+n])
		if err != nil {
			return nil, err
		}
		if rec.name != "\x00" && rec.name != "\x01" {
			out = append(out, rec)
		}
		off += n
	}
	return out, nil
}

func (im image) find(path string) (record, error) {
	cur, err := im.root()
	if err != nil {
		return record{}, err
	}
	for _, part := range strings.Split(strings.Trim(path, "/"), "/") {
		entries, err := im.list(cur)
		if err != nil {
			return record{}, err
		}
		found := false
		for _, e := range entries {
			if strings.EqualFold(e.name, part) {
				cur, found = e, true
				break
			}
		}
		if !found {
			return record{}, ErrNotFound
		}
	}
	return cur, nil
}

func (im image) read(path string, limit uint32) ([]byte, error) {
	rec, err := im.find(path)
	if err != nil {
		return nil, err
	}
	if rec.dir {
		return nil, fmt.Errorf("%s é diretório", path)
	}
	if rec.size > limit {
		return nil, fmt.Errorf("%s grande demais", path)
	}
	return im.readAt(rec.extent, int(rec.size))
}

// Read identifica o jogo de uma ISO aberta.
func Read(r io.ReaderAt) (Info, error) {
	im, err := open(r)
	if err != nil {
		return Info{}, err
	}
	if sfo, err := im.read("PSP_GAME/PARAM.SFO", 64<<10); err == nil {
		fields, err := ParseSFO(sfo)
		if err != nil {
			return Info{}, err
		}
		return Info{
			System: "psp", Serial: fields["DISC_ID"],
			Version: fields["DISC_VERSION"], Title: fields["TITLE"],
		}, nil
	} else if !errors.Is(err, ErrNotFound) {
		return Info{}, err
	}
	cnf, err := im.read("SYSTEM.CNF", 64<<10)
	if err != nil {
		return Info{}, err
	}
	if serial, err := PS2Serial(cnf); err == nil {
		return Info{System: "ps2", Serial: serial}, nil
	}
	serial, err := PS1Serial(cnf)
	if err != nil {
		return Info{}, err
	}
	return Info{System: "psx", Serial: serial}, nil
}

// ReadFile identifica o jogo de um arquivo .iso.
func ReadFile(path string) (Info, error) {
	f, err := os.Open(path)
	if err != nil {
		return Info{}, err
	}
	defer f.Close()
	return Read(f)
}

// ParseSFO lê as chaves de texto e número de um PARAM.SFO.
func ParseSFO(b []byte) (map[string]string, error) {
	if len(b) < 20 || !bytes.Equal(b[:4], []byte("\x00PSF")) {
		return nil, errors.New("PARAM.SFO inválido")
	}
	keys := binary.LittleEndian.Uint32(b[8:12])
	data := binary.LittleEndian.Uint32(b[12:16])
	count := binary.LittleEndian.Uint32(b[16:20])
	if count > 256 || 20+int(count)*16 > len(b) {
		return nil, errors.New("PARAM.SFO com entradas demais")
	}
	out := make(map[string]string, count)
	for i := range count {
		e := b[20+i*16 : 36+i*16]
		keyOff := keys + uint32(binary.LittleEndian.Uint16(e[0:2]))
		format := binary.LittleEndian.Uint16(e[2:4])
		length := binary.LittleEndian.Uint32(e[4:8])
		dataOff := data + binary.LittleEndian.Uint32(e[12:16])
		if int(keyOff) >= len(b) || int(dataOff)+int(length) > len(b) {
			return nil, errors.New("PARAM.SFO truncado")
		}
		key := cString(b[keyOff:])
		raw := b[dataOff : dataOff+length]
		switch format {
		case 0x0204, 0x0004: // texto UTF-8
			out[key] = cString(raw)
		case 0x0404: // inteiro
			if len(raw) >= 4 {
				out[key] = fmt.Sprint(binary.LittleEndian.Uint32(raw))
			}
		}
	}
	return out, nil
}

func cString(b []byte) string {
	if i := bytes.IndexByte(b, 0); i >= 0 {
		b = b[:i]
	}
	return string(b)
}

var (
	bootLine  = regexp.MustCompile(`(?im)^\s*BOOT2\s*=\s*cdrom0?:\\?([A-Z]{4})_(\d{3})\.(\d{2})`)
	boot1Line = regexp.MustCompile(`(?im)^\s*BOOT\s*=\s*cdrom:\\?([A-Z]{4})[_-](\d{3})\.(\d{2})`)
)

// PS1Serial tira o serial (SLUS-00067) da linha BOOT do SYSTEM.CNF de PS1.
func PS1Serial(cnf []byte) (string, error) {
	m := boot1Line.FindSubmatch(cnf)
	if m == nil {
		return "", errors.New("SYSTEM.CNF sem BOOT de PS1")
	}
	return fmt.Sprintf("%s-%s%s", m[1], m[2], m[3]), nil
}

// PS2Serial tira o serial (SLUS-21431) da linha BOOT2 do SYSTEM.CNF.
func PS2Serial(cnf []byte) (string, error) {
	m := bootLine.FindSubmatch(cnf)
	if m == nil {
		return "", errors.New("SYSTEM.CNF sem BOOT2 de PS2")
	}
	return fmt.Sprintf("%s-%s%s", m[1], m[2], m[3]), nil
}
