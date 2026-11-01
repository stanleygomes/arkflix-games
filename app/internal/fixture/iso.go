// Package fixture gera dados falsos (ISOs, bibliotecas, saves, RomM) pros
// testes em Go e pro modo de teste do servidor web (testes de ponta a ponta).
package fixture

import (
	"encoding/binary"
	"path"
	"sort"
	"strings"
)

const sector = 2048

type isoNode struct {
	name     string
	children map[string]*isoNode
	data     []byte
	dir      bool
	extent   uint32
}

// BuildISO monta uma ISO 9660 mínima com os arquivos dados (caminho -> conteúdo).
// Cada diretório cabe em um setor: suficiente pros testes.
func BuildISO(files map[string][]byte) []byte {
	root := &isoNode{dir: true, children: map[string]*isoNode{}}
	for p, data := range files {
		parts := strings.Split(strings.Trim(p, "/"), "/")
		cur := root
		for i, part := range parts {
			part = strings.ToUpper(part)
			next, ok := cur.children[part]
			if !ok {
				next = &isoNode{name: part, dir: i < len(parts)-1, children: map[string]*isoNode{}}
				cur.children[part] = next
			}
			if i == len(parts)-1 {
				next.data = data
			}
			cur = next
		}
	}

	next := uint32(18) // 16 = PVD, 17 = terminador
	var assign func(n *isoNode)
	assign = func(n *isoNode) {
		n.extent = next
		if n.dir {
			next++
			for _, c := range sorted(n.children) {
				assign(c)
			}
			return
		}
		next += max(1, uint32((len(n.data)+sector-1)/sector))
	}
	assign(root)

	img := make([]byte, int(next)*sector)
	var write func(n, parent *isoNode)
	write = func(n, parent *isoNode) {
		off := int(n.extent) * sector
		if !n.dir {
			copy(img[off:], n.data)
			return
		}
		buf := dirRecord("\x00", n.extent, sector, true)
		buf = append(buf, dirRecord("\x01", parent.extent, sector, true)...)
		for _, c := range sorted(n.children) {
			name, size := c.name, uint32(sector)
			if !c.dir {
				name, size = c.name+";1", uint32(len(c.data))
			}
			buf = append(buf, dirRecord(name, c.extent, size, c.dir)...)
		}
		copy(img[off:], buf)
		for _, c := range sorted(n.children) {
			write(c, n)
		}
	}
	write(root, root)

	pvd := img[16*sector:]
	pvd[0] = 1
	copy(pvd[1:6], "CD001")
	pvd[6] = 1
	copy(pvd[156:], dirRecord("\x00", root.extent, sector, true))
	term := img[17*sector:]
	term[0] = 255
	copy(term[1:6], "CD001")
	return img
}

func sorted(m map[string]*isoNode) []*isoNode {
	out := make([]*isoNode, 0, len(m))
	for _, v := range m {
		out = append(out, v)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].name < out[j].name })
	return out
}

func dirRecord(name string, extent, size uint32, dir bool) []byte {
	n := 33 + len(name)
	if n%2 == 1 {
		n++
	}
	b := make([]byte, n)
	b[0] = byte(n)
	binary.LittleEndian.PutUint32(b[2:], extent)
	binary.BigEndian.PutUint32(b[6:], extent)
	binary.LittleEndian.PutUint32(b[10:], size)
	binary.BigEndian.PutUint32(b[14:], size)
	if dir {
		b[25] = 2
	}
	b[28] = 1
	b[32] = byte(len(name))
	copy(b[33:], name)
	return b
}

// SFO monta um PARAM.SFO com chaves de texto.
func SFO(fields map[string]string) []byte {
	keys := make([]string, 0, len(fields))
	for k := range fields {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var keyTable, dataTable []byte
	entries := make([]byte, 0, 16*len(keys))
	for _, k := range keys {
		v := append([]byte(fields[k]), 0)
		e := make([]byte, 16)
		binary.LittleEndian.PutUint16(e[0:], uint16(len(keyTable)))
		binary.LittleEndian.PutUint16(e[2:], 0x0204)
		binary.LittleEndian.PutUint32(e[4:], uint32(len(v)))
		binary.LittleEndian.PutUint32(e[8:], uint32(len(v)))
		binary.LittleEndian.PutUint32(e[12:], uint32(len(dataTable)))
		entries = append(entries, e...)
		keyTable = append(keyTable, append([]byte(k), 0)...)
		dataTable = append(dataTable, v...)
	}
	keyStart := 20 + len(entries)
	dataStart := keyStart + len(keyTable)
	h := make([]byte, 20)
	copy(h, "\x00PSF")
	binary.LittleEndian.PutUint32(h[4:], 0x0101)
	binary.LittleEndian.PutUint32(h[8:], uint32(keyStart))
	binary.LittleEndian.PutUint32(h[12:], uint32(dataStart))
	binary.LittleEndian.PutUint32(h[16:], uint32(len(keys)))
	out := append(h, entries...)
	out = append(out, keyTable...)
	return append(out, dataTable...)
}

// PSPISO é uma ISO de PSP falsa com o DISC_ID e título dados.
func PSPISO(discID, title string) []byte {
	return BuildISO(map[string][]byte{
		"PSP_GAME/PARAM.SFO": SFO(map[string]string{
			"DISC_ID": discID, "DISC_VERSION": "1.00", "TITLE": title,
		}),
		"UMD_DATA.BIN": []byte(discID + "|0001"),
	})
}

// PS2ISO é uma ISO de PS2 falsa com o serial dado (SLUS-21431).
func PS2ISO(serial string) []byte {
	code := strings.ReplaceAll(serial, "-", "_")
	code = code[:8] + "." + code[8:]
	cnf := "BOOT2 = cdrom0:\\" + code + ";1\r\nVER = 1.00\r\nVMODE = NTSC\r\n"
	return BuildISO(map[string][]byte{
		"SYSTEM.CNF":         []byte(cnf),
		path.Join(code + ""): []byte("\x7fELF"),
	})
}

// RawBin converte uma ISO (setores de 2048) em .bin cru de CD (2352 por setor):
// modo 2 tem 24 bytes de cabeçalho, modo 1 tem 16.
func RawBin(iso []byte, mode2 bool) []byte {
	head := 16
	if mode2 {
		head = 24
	}
	n := len(iso) / sector
	out := make([]byte, n*2352)
	for i := range n {
		raw := out[i*2352:]
		copy(raw, []byte{0, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0})
		if mode2 {
			raw[15] = 2
		} else {
			raw[15] = 1
		}
		copy(raw[head:head+sector], iso[i*sector:(i+1)*sector])
	}
	return out
}

// PS1Bin é um .bin de PS1 falso (modo 2) com o serial dado (SLUS-00067).
func PS1Bin(serial string) []byte {
	code := strings.ReplaceAll(serial, "-", "_")
	code = code[:8] + "." + code[8:]
	cnf := "BOOT = cdrom:\\" + code + ";1\r\nTCB = 4\r\nEVENT = 10\r\n"
	return RawBin(BuildISO(map[string][]byte{"SYSTEM.CNF": []byte(cnf), code: []byte("PS-X EXE")}), true)
}
