package core

import (
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"sort"

	"github.com/stanleygomes/arkflix-games/app/internal/catalog"
	"github.com/stanleygomes/arkflix-games/app/internal/profiles"
	"github.com/stanleygomes/arkflix-games/app/internal/settings"
)

// StorageInfo é a tela de armazenamento: o disco dos jogos e quanto cada coisa
// ocupa, como a tela de armazenamento dos consoles.
type StorageInfo struct {
	Disk        string         `json:"disk"`  // pasta usada pra medir o disco (a de importação)
	Total       int64          `json:"total"` // -1: não deu pra medir
	Free        int64          `json:"free"`
	Games       int64          `json:"games"`
	GameCount   int            `json:"gameCount"`
	Saves       int64          `json:"saves"`
	App         int64          `json:"app"`   // configurações, perfis, listas, histórico
	Cache       int64          `json:"cache"` // capas baixadas e catálogo do servidor
	ByConsole   []ConsoleUsage `json:"byConsole"`
	SaveDirs    []PathUsage    `json:"saveDirs"`
	Remote      int64          `json:"remote"` // no servidor, ainda não baixado
	RemoteCount int            `json:"remoteCount"`
}

// ConsoleUsage: espaço dos jogos de um console.
type ConsoleUsage struct {
	System string `json:"system"`
	Name   string `json:"name"`
	Bytes  int64  `json:"bytes"`
	Count  int    `json:"count"`
	// Roots: as pastas da biblioteca onde estão (na ordem das configurações).
	Roots []string `json:"roots"`
}

// PathUsage: espaço de uma pasta (saves de um emulador...).
type PathUsage struct {
	Label string `json:"label"`
	Path  string `json:"path"`
	Bytes int64  `json:"bytes"`
}

// dirSize soma os arquivos da pasta (sem seguir links). Pasta que não existe = 0.
func dirSize(dir string, skip ...string) int64 {
	var total int64
	filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			for _, s := range skip {
				if path == s {
					return filepath.SkipDir
				}
			}
			return nil
		}
		if info, err := d.Info(); err == nil && info.Mode().IsRegular() {
			total += info.Size()
		}
		return nil
	})
	return total
}

func rootOrder(dirs []settings.LibraryDir, root string) int {
	for i, d := range dirs {
		if d.Path == root {
			return i
		}
	}
	return len(dirs)
}

// Storage mede o disco e o espaço de jogos, saves, dados do app e cache.
func (s *Service) Storage() StorageInfo {
	cfg := s.opt.Store.Settings()
	out := StorageInfo{Disk: cfg.ImportDir, ByConsole: []ConsoleUsage{}, SaveDirs: []PathUsage{}}
	if out.Disk == "" && len(cfg.LibraryDirs) > 0 {
		out.Disk = cfg.LibraryDirs[0].Path
	}
	out.Total, out.Free = diskUsage(out.Disk)

	byConsole := map[string]*ConsoleUsage{}
	s.mu.Lock()
	for _, e := range s.entries {
		if e.remote {
			out.Remote += e.game.Size
			out.RemoteCount++
			continue
		}
		c, ok := byConsole[e.game.System]
		if !ok {
			c = &ConsoleUsage{System: e.game.System, Name: e.game.System}
			if sys, found := catalog.Find(e.game.System); found {
				c.Name = sys.Name
			}
			byConsole[e.game.System] = c
		}
		c.Bytes += e.game.Size
		c.Count++
		if !slices.Contains(c.Roots, e.game.Root) {
			c.Roots = append(c.Roots, e.game.Root)
		}
		out.Games += e.game.Size
		out.GameCount++
	}
	s.mu.Unlock()
	for _, c := range byConsole {
		sort.SliceStable(c.Roots, func(i, j int) bool {
			return rootOrder(cfg.LibraryDirs, c.Roots[i]) < rootOrder(cfg.LibraryDirs, c.Roots[j])
		})
		out.ByConsole = append(out.ByConsole, *c)
	}
	sort.Slice(out.ByConsole, func(i, j int) bool { return out.ByConsole[i].Bytes > out.ByConsole[j].Bytes })

	// saves: as pastas dos emuladores (do perfil principal) e as dos perfis extras
	sd := cfg.Saves
	data := s.opt.Store.Dir()
	profilesDir := filepath.Join(data, "profiles")
	for _, p := range []PathUsage{
		{Label: "RetroArch (saves)", Path: sd.RetroArchSaves},
		{Label: "RetroArch (save states)", Path: sd.RetroArchStates},
		{Label: "PPSSPP (PSP)", Path: filepath.Dir(sd.PPSSPPSaveData)},
		{Label: "PCSX2 (memory cards)", Path: sd.PCSX2Memcards},
		{Label: "PCSX2 (save states)", Path: sd.PCSX2States},
		{Label: "Perfis extras", Path: profilesDir},
	} {
		if p.Path == "" {
			continue
		}
		p.Bytes = dirSize(p.Path)
		out.SaveDirs = append(out.SaveDirs, p)
		out.Saves += p.Bytes
	}

	// dados do app: tudo da pasta de dados menos os perfis (já contados nos saves)
	out.App = dirSize(data, profilesDir)
	if all, err := s.profiles.All(); err == nil && len(all) == 1 && all[0].ID == profiles.Main {
		if _, err := os.Stat(profilesDir); err != nil {
			out.SaveDirs = out.SaveDirs[:len(out.SaveDirs)-1] // sem perfis extras: some da lista
		}
	}
	out.Cache = dirSize(s.opt.Cache)
	return out
}
