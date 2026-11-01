// Package emucfg deixa a saída dos jogos igual em todos os emuladores:
//
//	Esc  / Select + botão de cima: menu do emulador (pausa, salvar, carregar)
//	F10  / Select + Start:          sai do jogo e volta pro app
//
// No RetroArch isso vai na config que o app acrescenta a cada jogo (package
// input). No PPSSPP e no PCSX2 não há como acrescentar config por linha de
// comando: aqui mexemos só nas chaves de atalho e de saída dos arquivos deles,
// e guardamos uma cópia do original (<arquivo>.arkflix-backup) na primeira vez.
package emucfg

import (
	"bufio"
	"errors"
	"os"
	"path/filepath"
	"strings"
)

// Ini é um .ini editado preservando tudo o que não for tocado (ordem,
// comentários, BOM). Chave repetida = lista (o PCSX2 usa assim).
type Ini struct {
	bom   bool
	lines []string
}

// ReadIni lê o arquivo (inexistente = vazio).
func ReadIni(path string) (*Ini, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return &Ini{}, nil
	}
	if err != nil {
		return nil, err
	}
	ini := &Ini{}
	text := string(data)
	if t, ok := strings.CutPrefix(text, "\uFEFF"); ok {
		ini.bom, text = true, t
	}
	sc := bufio.NewScanner(strings.NewReader(text))
	sc.Buffer(make([]byte, 1<<20), 1<<20)
	for sc.Scan() {
		ini.lines = append(ini.lines, strings.TrimRight(sc.Text(), "\r"))
	}
	return ini, sc.Err()
}

func section(line string) (string, bool) {
	l := strings.TrimSpace(line)
	if strings.HasPrefix(l, "[") && strings.HasSuffix(l, "]") {
		return l[1 : len(l)-1], true
	}
	return "", false
}

func key(line string) string {
	k, _, ok := strings.Cut(line, "=")
	if !ok || strings.HasPrefix(strings.TrimSpace(line), ";") || strings.HasPrefix(strings.TrimSpace(line), "#") {
		return ""
	}
	return strings.TrimSpace(k)
}

// span devolve o intervalo [início, fim) das linhas da seção (início = linha
// do cabeçalho); -1 se não existir.
func (i *Ini) span(sec string) (int, int) {
	start := -1
	for n, l := range i.lines {
		if s, ok := section(l); ok {
			if start >= 0 {
				return start, n
			}
			if s == sec {
				start = n
			}
		}
	}
	if start < 0 {
		return -1, -1
	}
	return start, len(i.lines)
}

// Get devolve os valores da chave (lista vazia se não houver).
func (i *Ini) Get(sec, k string) []string {
	start, end := i.span(sec)
	var out []string
	for n := start + 1; start >= 0 && n < end; n++ {
		if key(i.lines[n]) == k {
			_, v, _ := strings.Cut(i.lines[n], "=")
			out = append(out, strings.TrimSpace(v))
		}
	}
	return out
}

// Set troca todas as linhas da chave pelos valores (um por linha), no lugar
// da primeira; sem a chave, entra no fim da seção (criada se faltar).
func (i *Ini) Set(sec, k string, values ...string) {
	start, end := i.span(sec)
	if start < 0 {
		if len(i.lines) > 0 && strings.TrimSpace(i.lines[len(i.lines)-1]) != "" {
			i.lines = append(i.lines, "")
		}
		i.lines = append(i.lines, "["+sec+"]")
		start, end = len(i.lines)-1, len(i.lines)
	}
	fresh := make([]string, len(values))
	for n, v := range values {
		fresh[n] = k + " = " + v
	}
	var out []string
	placed := false
	for n, l := range i.lines {
		if n > start && n < end && key(l) == k {
			if !placed {
				out = append(out, fresh...)
				placed = true
			}
			continue
		}
		out = append(out, l)
	}
	if !placed {
		// fim da seção, antes das linhas em branco que a separam da próxima
		at := end
		for at > start+1 && strings.TrimSpace(out[at-1]) == "" {
			at--
		}
		out = append(out[:at], append(fresh, out[at:]...)...)
	}
	i.lines = out
}

func (i *Ini) String() string {
	s := strings.Join(i.lines, "\n") + "\n"
	if i.bom {
		s = "\uFEFF" + s
	}
	return s
}

// Write grava (atômico) e, na primeira vez, guarda o original ao lado.
func (i *Ini) Write(path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	old, err := os.ReadFile(path)
	if err == nil {
		if string(old) == i.String() {
			return nil
		}
		backup := path + ".arkflix-backup"
		if _, err := os.Stat(backup); errors.Is(err, os.ErrNotExist) {
			if err := os.WriteFile(backup, old, 0o644); err != nil {
				return err
			}
		}
	}
	tmp := path + ".arkflix-tmp"
	if err := os.WriteFile(tmp, []byte(i.String()), 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}
