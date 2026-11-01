// Package library varre as raízes de biblioteca (<raiz>/<console>/<jogo>) e
// monta a lista de jogos: título limpo, arquivo a abrir e de onde veio.
package library

import (
	"crypto/sha1"
	"encoding/hex"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/stanleygomes/arkflix-games/app/internal/catalog"
	"github.com/stanleygomes/arkflix-games/app/internal/settings"
)

// Game é um jogo encontrado no disco. Metadados (capa, resumo) entram depois.
type Game struct {
	ID       string `json:"id"`
	System   string `json:"system"`
	Title    string `json:"title"`
	FileName string `json:"fileName"` // nome no disco (arquivo ou pasta)
	Folder   string `json:"folder"`   // pasta do console como está no disco (megadrive, genesis...)
	Entry    string `json:"-"`        // caminho completo do arquivo ou pasta
	Launch   string `json:"-"`        // arquivo passado ao emulador
	Root     string `json:"root"`
	Local    bool   `json:"local"` // raiz gravável (neste PC)
	Size     int64  `json:"size"`
}

// ID estável: console + nome no disco. Igual em qualquer raiz, então o mesmo
// jogo no PC e no mount mantém favoritos e tempo de jogo.
func ID(system, fileName string) string {
	sum := sha1.Sum([]byte(system + "/" + strings.ToLower(fileName)))
	return hex.EncodeToString(sum[:6])
}

var (
	tagGroups  = regexp.MustCompile(`\s*[\(\[][^\)\]]*[\)\]]`)
	catalogNum = regexp.MustCompile(`^\d{3,5}\s+-\s+`)
	spaces     = regexp.MustCompile(`\s{2,}`)
)

// CleanTitle tira extensão, tags de região/versão e número de catálogo.
// "Super Mario World (U) [!].zip" -> "Super Mario World".
func CleanTitle(name string) string {
	ext := filepath.Ext(name)
	if len(ext) > 1 && len(ext) <= 5 && !strings.Contains(ext, " ") {
		name = strings.TrimSuffix(name, ext)
	}
	name = catalogNum.ReplaceAllString(name, "")
	clean := strings.TrimSpace(spaces.ReplaceAllString(tagGroups.ReplaceAllString(name, ""), " "))
	clean = strings.TrimRight(clean, " -_")
	if clean == "" {
		return strings.TrimSpace(name)
	}
	return clean
}

// launchFile escolhe o arquivo a abrir dentro de uma pasta de jogo
// (vários discos ou trilhas): .m3u > .cue > imagem única.
func launchFile(dir string, sys catalog.System) string {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return ""
	}
	byExt := map[string][]string{}
	for _, e := range entries {
		if e.IsDir() || strings.HasPrefix(e.Name(), ".") {
			continue
		}
		ext := strings.ToLower(filepath.Ext(e.Name()))
		byExt[ext] = append(byExt[ext], e.Name())
	}
	for _, ext := range []string{".m3u", ".cue", ".chd", ".pbp", ".iso", ".cso"} {
		if names := byExt[ext]; len(names) > 0 {
			sort.Strings(names)
			return filepath.Join(dir, names[0])
		}
	}
	for _, ext := range sys.Exts {
		if names := byExt[ext]; len(names) > 0 {
			sort.Strings(names)
			return filepath.Join(dir, names[0])
		}
	}
	return ""
}

func dirSize(path string) int64 {
	var total int64
	filepath.WalkDir(path, func(_ string, d os.DirEntry, err error) error {
		if err == nil && !d.IsDir() {
			if info, err := d.Info(); err == nil {
				total += info.Size()
			}
		}
		return nil
	})
	return total
}

// accept diz se o arquivo solto é jogo do console: extensão do console ou
// pacote (o RetroArch abre zip/7z). Temporários e arquivos ocultos ficam fora.
func accept(name string, sys catalog.System) bool {
	lower := strings.ToLower(name)
	if strings.HasPrefix(name, ".") || strings.HasSuffix(lower, ".part") {
		return false
	}
	ext := filepath.Ext(lower)
	if ext == ".zip" || ext == ".7z" {
		return true
	}
	for _, e := range sys.Exts {
		if e == ext {
			return true
		}
	}
	return false
}

// Scan varre todas as raízes. Jogo repetido (mesmo console + nome) vale o da
// primeira raiz da lista, normalmente a local.
func Scan(dirs []settings.LibraryDir) []Game {
	seen := map[string]bool{}
	var games []Game
	for _, root := range dirs {
		consoles, err := os.ReadDir(root.Path)
		if err != nil {
			continue // raiz ausente (mount desmontado): segue sem ela
		}
		for _, c := range consoles {
			sys, ok := catalog.Find(c.Name())
			if !ok || !c.IsDir() {
				continue
			}
			base := filepath.Join(root.Path, c.Name())
			items, err := os.ReadDir(base)
			if err != nil {
				continue
			}
			for _, it := range items {
				if strings.HasPrefix(it.Name(), ".") || (!it.IsDir() && !accept(it.Name(), sys)) {
					continue
				}
				g := Game{
					System: sys.Slug, FileName: it.Name(), Folder: c.Name(), Title: CleanTitle(it.Name()),
					Entry: filepath.Join(base, it.Name()), Root: root.Path, Local: root.Writable,
				}
				if it.IsDir() {
					g.Launch = launchFile(g.Entry, sys)
					g.Size = dirSize(g.Entry)
				} else {
					g.Launch = g.Entry
					if info, err := it.Info(); err == nil {
						g.Size = info.Size()
					}
				}
				if g.Launch == "" {
					continue
				}
				g.ID = ID(sys.Slug, g.FileName)
				if seen[g.ID] {
					continue
				}
				seen[g.ID] = true
				games = append(games, g)
			}
		}
	}
	sort.SliceStable(games, func(i, j int) bool {
		if games[i].System != games[j].System {
			return systemOrder(games[i].System) < systemOrder(games[j].System)
		}
		return strings.ToLower(games[i].Title) < strings.ToLower(games[j].Title)
	})
	return games
}

func systemOrder(slug string) int {
	for i, s := range catalog.Systems {
		if s.Slug == slug {
			return i
		}
	}
	return len(catalog.Systems)
}

// Stem é o nome do jogo sem extensão (pasta de disco fica como está).
func (g Game) Stem() string {
	if g.Launch != g.Entry {
		return g.FileName
	}
	return strings.TrimSuffix(g.FileName, filepath.Ext(g.FileName))
}

// LocalSource é a fonte de jogos das pastas locais. Um servidor no futuro
// implementa a mesma interface (core.GameSource).
type LocalSource struct {
	Dirs func() []settings.LibraryDir
}

// Scan varre as pastas configuradas.
func (l LocalSource) Scan() []Game { return Scan(l.Dirs()) }
