// Package settings guarda as preferências do app em JSON (settings.json) e o
// estado de uso (favoritos, recentes, tempo de jogo) em state.json.
package settings

import (
	"encoding/json"
	"errors"
	"maps"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// LibraryDir é uma raiz de biblioteca: <Path>/<console>/<jogos>.
type LibraryDir struct {
	Path     string `json:"path"`
	Label    string `json:"label"`
	Writable bool   `json:"writable"`
}

// Accessibility são as opções de leitura, movimento e som.
type Accessibility struct {
	FontScale    float64 `json:"fontScale"`
	HighContrast bool    `json:"highContrast"`
	ReduceMotion bool    `json:"reduceMotion"`
	Scanlines    bool    `json:"scanlines"`
	PixelTitles  bool    `json:"pixelTitles"`
	Sounds       bool    `json:"sounds"`
	SoundVolume  int     `json:"soundVolume"`
	Music        bool    `json:"music"`
	MusicVolume  int     `json:"musicVolume"`
}

// Covers liga o download de capas do acervo aberto do libretro.
type Covers struct {
	Enabled bool   `json:"enabled"`
	Images  string `json:"images"` // servidor das imagens
	DB      string `json:"db"`     // servidor do banco (DATs)
}

// UI guarda preferências da interface: ordem de cada lista (console,
// coleção) e filtros.
type UI struct {
	Sort    map[string]string   `json:"sort"`
	Filters map[string][]string `json:"filters"`
}

// SaveDirs são as pastas de save de cada emulador (as dos Flatpaks, que os
// lançadores compartilham).
type SaveDirs struct {
	RetroArchSaves  string `json:"retroarchSaves"`
	RetroArchStates string `json:"retroarchStates"`
	PPSSPPSaveData  string `json:"ppssppSaveData"`
	PPSSPPStates    string `json:"ppssppStates"`
	PCSX2Memcards   string `json:"pcsx2Memcards"`
	PCSX2States     string `json:"pcsx2States"`
}

// Settings é tudo que a tela de configurações mexe.
type Settings struct {
	LibraryDirs []LibraryDir  `json:"libraryDirs"`
	ImportDir   string        `json:"importDir"`
	EmulatorBin string        `json:"emulatorBin"`
	CoresDir    string        `json:"coresDir"`
	Fullscreen  bool          `json:"fullscreen"`
	ESDEDir     string        `json:"esdeDir"` // capas e descrições locais (gamelists do ES-DE)
	Covers      Covers        `json:"covers"`
	UI          UI            `json:"ui"`
	A11y        Accessibility `json:"a11y"`
	Saves       SaveDirs      `json:"saves"`
}

// Defaults usa a pasta do usuário. bin e cores vêm de quem inicia o app.
func Defaults(home, bin, cores string) Settings {
	varApp := filepath.Join(home, ".var", "app")
	ra := filepath.Join(varApp, "org.libretro.RetroArch", "config", "retroarch")
	psp := filepath.Join(varApp, "org.ppsspp.PPSSPP", "config", "ppsspp", "PSP")
	ps2 := filepath.Join(varApp, "net.pcsx2.PCSX2", "config", "PCSX2")
	return Settings{
		LibraryDirs: []LibraryDir{
			{Path: filepath.Join(home, "Games", "roms"), Label: "Jogos", Writable: true},
		},
		ImportDir:   filepath.Join(home, "Games", "roms"),
		EmulatorBin: bin,
		CoresDir:    cores,
		Fullscreen:  true,
		ESDEDir:     filepath.Join(home, "ES-DE"),
		Covers: Covers{
			Images: "https://thumbnails.libretro.com",
			DB:     "https://raw.githubusercontent.com/libretro/libretro-database/master/metadat",
		},
		A11y: Accessibility{
			FontScale: 1, Scanlines: true, PixelTitles: true,
			Sounds: true, SoundVolume: 60, Music: true, MusicVolume: 25,
		},
		Saves: SaveDirs{
			RetroArchSaves:  filepath.Join(ra, "saves"),
			RetroArchStates: filepath.Join(ra, "states"),
			PPSSPPSaveData:  filepath.Join(psp, "SAVEDATA"),
			PPSSPPStates:    filepath.Join(psp, "PPSSPP_STATE"),
			PCSX2Memcards:   filepath.Join(ps2, "memcards"),
			PCSX2States:     filepath.Join(ps2, "sstates"),
		},
	}
}

// Validate corrige valores fora do intervalo em vez de falhar: a tela sempre abre.
func (s *Settings) Validate() {
	if s.UI.Sort == nil {
		s.UI.Sort = map[string]string{}
	}
	if s.UI.Filters == nil {
		s.UI.Filters = map[string][]string{}
	}
	if s.A11y.FontScale < 0.75 || s.A11y.FontScale > 2 {
		s.A11y.FontScale = 1
	}
	s.A11y.SoundVolume = clamp(s.A11y.SoundVolume, 0, 100)
	s.A11y.MusicVolume = clamp(s.A11y.MusicVolume, 0, 100)
}

func clamp(v, lo, hi int) int {
	return max(lo, min(hi, v))
}

// State é o que muda jogando: favoritos, últimos jogados, tempo de jogo.
type State struct {
	Favorites map[string]bool  `json:"favorites"`
	LastPlay  map[string]int64 `json:"lastPlay"`
	PlayTime  map[string]int64 `json:"playTime"`
	Seen      map[string]int64 `json:"seen"`
}

// Store lê e grava os dois arquivos com escrita atômica (tmp + rename).
type Store struct {
	dir      string
	defaults Settings
	mu       sync.Mutex
	settings Settings
	state    State
}

// Open carrega o que existir em dir; o que faltar vem dos padrões.
func Open(dir string, defaults Settings) (*Store, error) {
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, err
	}
	st := &Store{dir: dir, defaults: defaults, settings: defaults}
	if err := readJSON(filepath.Join(dir, "settings.json"), &st.settings); err != nil {
		return nil, err
	}
	st.settings.Validate()
	if err := readJSON(filepath.Join(dir, "state.json"), &st.state); err != nil {
		return nil, err
	}
	st.state.init()
	return st, nil
}

func (s *State) init() {
	if s.Favorites == nil {
		s.Favorites = map[string]bool{}
	}
	if s.LastPlay == nil {
		s.LastPlay = map[string]int64{}
	}
	if s.PlayTime == nil {
		s.PlayTime = map[string]int64{}
	}
	if s.Seen == nil {
		s.Seen = map[string]int64{}
	}
}

func readJSON(path string, v any) error {
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	return json.Unmarshal(data, v)
}

func writeJSON(path string, v any) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

// Dir é a pasta de dados do app (cache, configs geradas).
func (s *Store) Dir() string { return s.dir }

// Settings devolve uma cópia.
func (s *Store) Settings() Settings {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := s.settings
	out.LibraryDirs = append([]LibraryDir(nil), s.settings.LibraryDirs...)
	return out
}

// Save valida e grava as configurações.
func (s *Store) Save(next Settings) (Settings, error) {
	next.Validate()
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := writeJSON(filepath.Join(s.dir, "settings.json"), next); err != nil {
		return s.settings, err
	}
	s.settings = next
	return next, nil
}

// Reset volta tudo ao padrão.
func (s *Store) Reset() (Settings, error) { return s.Save(s.defaults) }

// UpdateState muda o estado de uso e grava.
func (s *Store) UpdateState(fn func(*State)) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	fn(&s.state)
	return writeJSON(filepath.Join(s.dir, "state.json"), s.state)
}

// State devolve uma cópia (dá pra ler sem segurar o lock).
func (s *Store) State() State {
	s.mu.Lock()
	defer s.mu.Unlock()
	return State{
		Favorites: maps.Clone(s.state.Favorites),
		LastPlay:  maps.Clone(s.state.LastPlay),
		PlayTime:  maps.Clone(s.state.PlayTime),
		Seen:      maps.Clone(s.state.Seen),
	}
}

// RecordPlay soma o tempo de jogo e marca como jogado agora.
func (s *Store) RecordPlay(id string, d time.Duration, now time.Time) error {
	return s.UpdateState(func(st *State) {
		st.LastPlay[id] = now.Unix()
		st.PlayTime[id] += int64(d.Seconds())
	})
}
