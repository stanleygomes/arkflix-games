// Package isoinfo lê o mínimo de uma imagem ISO 9660 pra identificar o jogo:
// PSP (PSP_GAME/PARAM.SFO: DISC_ID, título) e PS2 (SYSTEM.CNF: serial).
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

type image struct{ r io.ReaderAt }

func (im image) root() (record, error) {
	pvd := make([]byte, sector)
	if _, err := im.r.ReadAt(pvd, 16*sector); err != nil {
		return record{}, fmt.Errorf("sem descritor ISO 9660: %w", err)
	}
	if pvd[0] != 1 || string(pvd[1:6]) != "CD001" {
		return record{}, errors.New("não é uma imagem ISO 9660")
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
	data := make([]byte, dir.size)
	if _, err := im.r.ReadAt(data, int64(dir.extent)*sector); err != nil {
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
	buf := make([]byte, rec.size)
	_, err = im.r.ReadAt(buf, int64(rec.extent)*sector)
	return buf, err
}

// Read identifica o jogo de uma ISO aberta.
func Read(r io.ReaderAt) (Info, error) {
	im := image{r}
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
	serial, err := PS2Serial(cnf)
	if err != nil {
		return Info{}, err
	}
	return Info{System: "ps2", Serial: serial}, nil
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

var bootLine = regexp.MustCompile(`(?im)^\s*BOOT2\s*=\s*cdrom0?:\\?([A-Z]{4})_(\d{3})\.(\d{2})`)

// PS2Serial tira o serial (SLUS-21431) da linha BOOT2 do SYSTEM.CNF.
func PS2Serial(cnf []byte) (string, error) {
	m := bootLine.FindSubmatch(cnf)
	if m == nil {
		return "", errors.New("SYSTEM.CNF sem BOOT2 de PS2")
	}
	return fmt.Sprintf("%s-%s%s", m[1], m[2], m[3]), nil
}
