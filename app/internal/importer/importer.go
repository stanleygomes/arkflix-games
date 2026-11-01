// Package importer leva ROMs de qualquer pasta pra biblioteca local: detecta o
// console (extensão, conteúdo do pacote, ISO), valida e copia ou extrai.
package importer

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/stanleygomes/arkflix-games/app/internal/archive"
	"github.com/stanleygomes/arkflix-games/app/internal/catalog"
	"github.com/stanleygomes/arkflix-games/app/internal/isoinfo"
	"github.com/stanleygomes/arkflix-games/app/internal/library"
)

// Item é um candidato a importação, já analisado.
type Item struct {
	Source     string   `json:"source"`
	Name       string   `json:"name"`       // nome final na biblioteca
	Title      string   `json:"title"`      // título limpo, pra mostrar
	System     string   `json:"system"`     // detectado (vazio = escolher)
	Candidates []string `json:"candidates"` // consoles possíveis
	Extract    bool     `json:"extract"`    // pacote com disco: vira pasta
	Size       int64    `json:"size"`
	Problem    string   `json:"problem,omitempty"`
}

var unsupported = map[string]string{
	".gdi":  "Dreamcast não é suportado",
	".cdi":  "Dreamcast não é suportado",
	".xci":  "Switch não é suportado",
	".nsp":  "Switch não é suportado",
	".3ds":  "3DS não é suportado",
	".wbfs": "Wii não é suportado",
	".rvz":  "GameCube/Wii não é suportado",
}

var discExts = map[string]bool{".cue": true, ".bin": true, ".iso": true, ".chd": true, ".m3u": true, ".img": true, ".pbp": true, ".cso": true}

func slugs(systems []catalog.System) []string {
	out := make([]string, len(systems))
	for i, s := range systems {
		out[i] = s.Slug
	}
	return out
}

// Analyze inspeciona cada caminho (arquivo ou pasta) sem mexer em nada.
func Analyze(paths []string) []Item {
	var out []Item
	for _, p := range paths {
		out = append(out, analyze(p))
	}
	return out
}

func analyze(path string) Item {
	it := Item{Source: path, Name: filepath.Base(path), Title: library.CleanTitle(filepath.Base(path))}
	info, err := os.Stat(path)
	if err != nil {
		it.Problem = "arquivo não encontrado"
		return it
	}
	if info.IsDir() {
		return analyzeDir(it)
	}
	it.Size = info.Size()
	ext := strings.ToLower(filepath.Ext(path))
	if why, ok := unsupported[ext]; ok {
		it.Problem = why
		return it
	}
	kind, err := archive.Detect(path)
	if err != nil {
		it.Problem = err.Error()
		return it
	}
	if kind != archive.None {
		return analyzeArchive(it)
	}
	switch ext {
	case ".iso":
		if gi, err := isoinfo.ReadFile(path); err == nil {
			it.System, it.Candidates = gi.System, []string{gi.System}
			if gi.Title != "" {
				it.Title = gi.Title
			}
		} else {
			it.Candidates = []string{"psp", "ps2"}
		}
		return it
	case ".cue":
		it.System, it.Candidates = "psx", []string{"psx"}
		it.Name = library.CleanTitle(it.Name)
		it.Extract = true // vira pasta com o .cue e os .bin
		return it
	}
	cands := catalog.ByExtension(path)
	it.Candidates = slugs(cands)
	if len(cands) == 1 {
		it.System = cands[0].Slug
	}
	if len(cands) == 0 {
		it.Problem = "extensão desconhecida: escolha o console"
		it.Candidates = slugs(catalog.Systems)
	}
	return it
}

func analyzeArchive(it Item) Item {
	entries, err := archive.List(it.Source)
	if err != nil {
		it.Problem = err.Error()
		return it
	}
	found := map[string]bool{}
	hasDisc, known := false, false
	for _, e := range entries {
		ext := strings.ToLower(filepath.Ext(e.Name))
		if why, ok := unsupported[ext]; ok {
			it.Problem = why
			return it
		}
		if discExts[ext] {
			hasDisc = true
		}
		for _, s := range catalog.ByExtension(e.Name) {
			if s.Slug == "arcade" {
				continue
			}
			found[s.Slug] = true
			known = true
		}
	}
	for s := range found {
		it.Candidates = append(it.Candidates, s)
	}
	sort.Strings(it.Candidates)
	switch {
	case hasDisc && found["psx"]:
		it.System, it.Extract = "psx", true
		it.Candidates = []string{"psx"}
	case hasDisc:
		it.Extract = true
		if len(it.Candidates) == 1 {
			it.System = it.Candidates[0]
		} else {
			it.Candidates = []string{"psx", "ps2", "psp"}
		}
	case len(it.Candidates) == 1:
		it.System = it.Candidates[0]
	case !known && len(entries) > 0:
		// sem extensão de console conhecida: placa de arcade (pacman.6e, ...)
		it.System, it.Candidates = "arcade", []string{"arcade"}
	case len(entries) == 0:
		it.Problem = "pacote vazio"
	}
	if it.Extract {
		it.Name = library.CleanTitle(it.Name)
	}
	return it
}

func analyzeDir(it Item) Item {
	var names []string
	filepath.WalkDir(it.Source, func(p string, d os.DirEntry, err error) error {
		if err == nil && !d.IsDir() {
			names = append(names, p)
			if info, err := d.Info(); err == nil {
				it.Size += info.Size()
			}
		}
		return nil
	})
	for _, n := range names {
		ext := strings.ToLower(filepath.Ext(n))
		if ext == ".cue" || ext == ".m3u" {
			it.System, it.Candidates = "psx", []string{"psx"}
			return it
		}
		if ext == ".iso" {
			if gi, err := isoinfo.ReadFile(n); err == nil {
				it.System, it.Candidates = gi.System, []string{gi.System}
				return it
			}
		}
	}
	if sys, ok := catalog.Find(filepath.Base(it.Source)); ok {
		it.Problem = "pasta de console inteira: importe os arquivos de dentro"
		it.Candidates = []string{sys.Slug}
		return it
	}
	it.Problem = "pasta sem jogo reconhecível"
	return it
}

// Result diz o que aconteceu com cada item importado.
type Result struct {
	Item
	Dest string `json:"dest,omitempty"`
	OK   bool   `json:"ok"`
}

// Import copia/extrai os itens pra <dest>/<console>/. Item sem console ou com
// problema é pulado com o motivo; nada é sobrescrito.
func Import(items []Item, dest string) []Result {
	out := make([]Result, 0, len(items))
	for _, it := range items {
		r := Result{Item: it}
		switch {
		case it.Problem != "":
		case it.System == "":
			r.Problem = "escolha o console"
		default:
			if _, ok := catalog.Find(it.System); !ok {
				r.Problem = "console desconhecido: " + it.System
				break
			}
			r.Dest, r.Problem = place(it, filepath.Join(dest, it.System))
			r.OK = r.Problem == ""
		}
		out = append(out, r)
	}
	return out
}

func place(it Item, dir string) (string, string) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err.Error()
	}
	target := filepath.Join(dir, it.Name)
	if _, err := os.Stat(target); err == nil {
		return "", "já existe na biblioteca"
	}
	info, err := os.Stat(it.Source)
	if err != nil {
		return "", "arquivo não encontrado"
	}
	tmp, err := os.MkdirTemp(dir, ".import-")
	if err != nil {
		return "", err.Error()
	}
	defer os.RemoveAll(tmp)
	staged := filepath.Join(tmp, it.Name)
	switch {
	case info.IsDir():
		err = copyDir(it.Source, staged)
	case strings.EqualFold(filepath.Ext(it.Source), ".cue"):
		err = copyCue(it.Source, staged)
	case it.Extract:
		err = extractDisc(it.Source, staged)
	default:
		err = copyFile(it.Source, staged)
	}
	if err != nil {
		return "", err.Error()
	}
	if err := FixCues(staged); err != nil {
		return "", err.Error()
	}
	// rename só no fim: a biblioteca nunca vê cópia pela metade
	if err := os.Rename(staged, target); err != nil {
		return "", err.Error()
	}
	return target, ""
}

// extractDisc extrai o pacote; se sobrar uma pasta só lá dentro, sobe o conteúdo.
func extractDisc(src, staged string) error {
	if err := archive.Extract(src, staged); err != nil {
		return err
	}
	entries, err := os.ReadDir(staged)
	if err != nil {
		return err
	}
	if len(entries) == 1 && entries[0].IsDir() {
		inner := filepath.Join(staged, entries[0].Name())
		tmp := staged + ".inner"
		if err := os.Rename(inner, tmp); err != nil {
			return err
		}
		if err := os.Remove(staged); err != nil {
			return err
		}
		return os.Rename(tmp, staged)
	}
	return nil
}

var cueFile = regexp.MustCompile(`(?i)^(\s*FILE\s+)"?([^"]+?)"?(\s+\w+\s*)$`)

// copyCue copia o .cue e os arquivos que ele cita pra uma pasta nova.
func copyCue(cue, dir string) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	f, err := os.Open(cue)
	if err != nil {
		return err
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		if m := cueFile.FindStringSubmatch(sc.Text()); m != nil {
			name := filepath.Base(strings.ReplaceAll(m[2], `\`, "/"))
			src := filepath.Join(filepath.Dir(cue), name)
			if err := copyFile(src, filepath.Join(dir, name)); err != nil {
				return fmt.Errorf("o .cue cita %s, que não está junto: %w", name, err)
			}
		}
	}
	return copyFile(cue, filepath.Join(dir, filepath.Base(cue)))
}

// FixCues reescreve as linhas FILE dos .cue pra só o nome do arquivo (cue
// gerado no Windows traz caminho absoluto e quebra o emulador).
func FixCues(dir string) error {
	info, err := os.Stat(dir)
	if err != nil || !info.IsDir() {
		return nil
	}
	cues, _ := filepath.Glob(filepath.Join(dir, "*.[cC][uU][eE]"))
	for _, cue := range cues {
		data, err := os.ReadFile(cue)
		if err != nil {
			return err
		}
		lines := strings.Split(string(data), "\n")
		changed := false
		for i, line := range lines {
			m := cueFile.FindStringSubmatch(strings.TrimRight(line, "\r"))
			if m == nil {
				continue
			}
			name := filepath.Base(strings.ReplaceAll(m[2], `\`, "/"))
			fixed := fmt.Sprintf(`%s"%s"%s`, m[1], name, m[3])
			if strings.HasSuffix(line, "\r") {
				fixed += "\r"
			}
			if fixed != line {
				lines[i], changed = fixed, true
			}
		}
		if changed {
			if err := os.WriteFile(cue, []byte(strings.Join(lines, "\n")), 0o644); err != nil {
				return err
			}
		}
	}
	return nil
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		return err
	}
	return out.Close()
}

func copyDir(src, dst string) error {
	return filepath.WalkDir(src, func(p string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(src, p)
		target := filepath.Join(dst, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		if d.Type()&os.ModeSymlink != 0 {
			return errors.New("link simbólico na pasta: copie o arquivo de verdade")
		}
		return copyFile(p, target)
	})
}
