// Package meta dá nome, descrição, ano, gêneros e capa dos jogos a partir de
// arquivos locais: os gamelist.xml e a mídia do ES-DE, ou uma pasta covers/
// ao lado das ROMs. Implementa core.MetaProvider; um servidor (RomM, por
// exemplo) entra depois com outra implementação da mesma interface.
package meta

import (
	"encoding/xml"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"github.com/stanleygomes/arkflix-games/app/internal/library"
)

// Info são os metadados de um jogo. Cover é um caminho local.
type Info struct {
	Name    string   `json:"name"`
	Summary string   `json:"summary"`
	Year    int      `json:"year"`
	Genres  []string `json:"genres"`
	Players string   `json:"players"`
	// Developer: desenvolvedora (gamelist do ES-DE ou edição).
	Developer string `json:"developer"`
	Cover     string `json:"-"`
	// Snap (print do jogo) e TitleShot (tela de título): caminhos locais.
	Snap      string `json:"-"`
	TitleShot string `json:"-"`
	// Manual em PDF (pasta manuals/ ou mídia do ES-DE).
	Manual string `json:"-"`
}

type gameList struct {
	Games []struct {
		Path        string `xml:"path"`
		Name        string `xml:"name"`
		Desc        string `xml:"desc"`
		ReleaseDate string `xml:"releasedate"`
		Genre       string `xml:"genre"`
		Players     string `xml:"players"`
		Developer   string `xml:"developer"`
	} `xml:"game"`
}

// Local lê os metadados do disco, com cache por pasta de console.
type Local struct {
	ESDEDir func() string

	mu    sync.Mutex
	lists map[string]map[string]Info // pasta do console -> nome do arquivo -> info
}

var imageExts = []string{".png", ".jpg", ".jpeg", ".webp"}

// Refresh esquece o cache (gamelist ou capa nova).
func (l *Local) Refresh() {
	l.mu.Lock()
	l.lists = nil
	l.mu.Unlock()
}

func (l *Local) gamelist(folder string) map[string]Info {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.lists == nil {
		l.lists = map[string]map[string]Info{}
	}
	if m, ok := l.lists[folder]; ok {
		return m
	}
	m := map[string]Info{}
	l.lists[folder] = m
	dir := l.esde()
	if dir == "" {
		return m
	}
	data, err := os.ReadFile(filepath.Join(dir, "gamelists", folder, "gamelist.xml"))
	if err != nil {
		return m
	}
	var gl gameList
	if xml.Unmarshal(data, &gl) != nil {
		return m
	}
	for _, g := range gl.Games {
		info := Info{Name: strings.TrimSpace(g.Name), Summary: strings.TrimSpace(g.Desc), Players: strings.TrimSpace(g.Players), Developer: strings.TrimSpace(g.Developer)}
		if len(g.ReleaseDate) >= 4 {
			info.Year, _ = strconv.Atoi(g.ReleaseDate[:4])
		}
		for _, genre := range strings.Split(g.Genre, ",") {
			if genre = strings.TrimSpace(genre); genre != "" {
				info.Genres = append(info.Genres, genre)
			}
		}
		m[strings.ToLower(filepath.Base(g.Path))] = info
	}
	return m
}

func (l *Local) esde() string {
	if l.ESDEDir == nil {
		return ""
	}
	return l.ESDEDir()
}

func findImage(dir, stem string) string {
	for _, ext := range imageExts {
		path := filepath.Join(dir, stem+ext)
		if info, err := os.Stat(path); err == nil && !info.IsDir() {
			return path
		}
	}
	return ""
}

// Lookup junta o que achar: gamelist do ES-DE (pela pasta do console no disco
// e pelo nome padrão) e a capa local.
func (l *Local) Lookup(g library.Game) Info {
	var info Info
	for _, folder := range []string{g.Folder, g.System} {
		if found, ok := l.gamelist(folder)[strings.ToLower(g.FileName)]; ok {
			info = found
			break
		}
	}
	stem := g.Stem()
	dirs := []string{filepath.Join(g.Root, g.Folder, "covers")}
	if esde := l.esde(); esde != "" {
		for _, folder := range []string{g.Folder, g.System} {
			dirs = append(dirs,
				filepath.Join(esde, "downloaded_media", folder, "covers"),
				filepath.Join(esde, "downloaded_media", folder, "miximages"),
				filepath.Join(esde, "downloaded_media", folder, "screenshots"))
		}
	}
	for _, dir := range dirs {
		if cover := findImage(dir, stem); cover != "" {
			info.Cover = cover
			break
		}
	}
	info.Snap = findImage(filepath.Join(g.Root, g.Folder, "screenshots"), stem)
	manuals := []string{filepath.Join(g.Root, g.Folder, "manuals")}
	if esde := l.esde(); esde != "" {
		manuals = append(manuals, filepath.Join(esde, "downloaded_media", g.Folder, "manuals"), filepath.Join(esde, "downloaded_media", g.System, "manuals"))
	}
	for _, dir := range manuals {
		path := filepath.Join(dir, stem+".pdf")
		if st, err := os.Stat(path); err == nil && !st.IsDir() {
			info.Manual = path
			break
		}
	}
	return info
}
