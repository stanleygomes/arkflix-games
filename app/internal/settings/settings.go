// Package settings guarda as preferências do app em JSON (settings.json) e o
// estado de uso (favoritos, recentes, tempo de jogo) em state.json.
package settings

import (
	"encoding/json"
	"errors"
	"maps"
	"os"
	"path/filepath"
	"runtime"
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
	Videos       bool    `json:"videos"`     // vídeo do jogo no destaque
	VideoSound   bool    `json:"videoSound"` // com som (padrão: mudo)
}

// Emulators: caminhos dos emuladores. Vazio = padrão do sistema (Flatpak no
// Linux; no Windows e no macOS, os lugares onde os instaladores oficiais põem).
type Emulators struct {
	RetroArch string `json:"retroarch"`
	Cores     string `json:"cores"`
	PPSSPP    string `json:"ppsspp"`
	PCSX2     string `json:"pcsx2"`
}

// Screensaver é o protetor de tela: liga depois de Minutes sem entrada.
type Screensaver struct {
	Enabled bool   `json:"enabled"`
	Minutes int    `json:"minutes"`
	Mode    string `json:"mode"` // "snake" (cobrinha) ou "showcase" (vitrine)
	Rest    int    `json:"rest"` // minutos de protetor até o descanso (tela preta); 0 = nunca
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

// Downloads: jogos do servidor. LimitKBps = velocidade máxima (0 = sem
// limite); NightOnly = a fila só anda entre 0h e 6h (hora local).
type Downloads struct {
	LimitKBps int  `json:"limitKBps"`
	NightOnly bool `json:"nightOnly"`
}

// Settings é tudo que a tela de configurações mexe.
type Settings struct {
	LibraryDirs []LibraryDir  `json:"libraryDirs"`
	ImportDir   string        `json:"importDir"`
	Fullscreen  bool          `json:"fullscreen"`
	ESDEDir     string        `json:"esdeDir"` // capas e descrições locais (gamelists do ES-DE)
	Covers      Covers        `json:"covers"`
	UI          UI            `json:"ui"`
	Emulators   Emulators     `json:"emulators"`
	A11y        Accessibility `json:"a11y"`
	Screensaver Screensaver   `json:"screensaver"`
	Saves       SaveDirs      `json:"saves"`
	Downloads   Downloads     `json:"downloads"`
	// AutoUpdate: baixa a versão nova sozinho e instala ao fechar o app.
	AutoUpdate bool `json:"autoUpdate"`
}

// Defaults usa a pasta do usuário.
func Defaults(home string) Settings { return DefaultsFor(runtime.GOOS, home) }

// emulatorDirs: onde cada emulador guarda config e saves por padrão, em cada
// sistema (Linux: Flatpaks; Windows: instalador do RetroArch e Documentos;
// macOS: Application Support).
func emulatorDirs(goos, home string) (ra, psp, ps2 string) {
	switch goos {
	case "windows":
		docs := filepath.Join(home, "Documents")
		return filepath.Join("C:"+string(filepath.Separator), "RetroArch-Win64"), filepath.Join(docs, "PPSSPP", "PSP"), filepath.Join(docs, "PCSX2")
	case "darwin":
		support := filepath.Join(home, "Library", "Application Support")
		return filepath.Join(support, "RetroArch"), filepath.Join(support, "PPSSPP", "PSP"), filepath.Join(support, "PCSX2")
	}
	varApp := filepath.Join(home, ".var", "app")
	return filepath.Join(varApp, "org.libretro.RetroArch", "config", "retroarch"),
		filepath.Join(varApp, "org.ppsspp.PPSSPP", "config", "ppsspp", "PSP"),
		filepath.Join(varApp, "net.pcsx2.PCSX2", "config", "PCSX2")
}

// DefaultsFor é o padrão de um sistema (goos do Go: linux, windows, darwin).
func DefaultsFor(goos, home string) Settings {
	ra, psp, ps2 := emulatorDirs(goos, home)
	return Settings{
		LibraryDirs: []LibraryDir{
			{Path: filepath.Join(home, "Games", "roms"), Label: "Jogos", Writable: true},
		},
		ImportDir:  filepath.Join(home, "Games", "roms"),
		Fullscreen: true,
		AutoUpdate: true,
		ESDEDir:    filepath.Join(home, "ES-DE"),
		Covers: Covers{
			Images: "https://thumbnails.libretro.com",
			DB:     "https://raw.githubusercontent.com/libretro/libretro-database/master/metadat",
		},
		A11y: Accessibility{
			FontScale: 1, Scanlines: true, PixelTitles: true,
			Sounds: true, SoundVolume: 60, Music: true, MusicVolume: 25,
			Videos: true,
		},
		Screensaver: Screensaver{Enabled: true, Minutes: 5, Mode: "snake", Rest: 15},
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
	if s.Screensaver.Minutes <= 0 {
		s.Screensaver.Minutes = 5
	}
	s.Screensaver.Minutes = min(s.Screensaver.Minutes, 60)
	if s.Screensaver.Mode != "snake" && s.Screensaver.Mode != "showcase" {
		s.Screensaver.Mode = "snake"
	}
	s.Screensaver.Rest = clamp(s.Screensaver.Rest, 0, 240)
	s.Downloads.LimitKBps = clamp(s.Downloads.LimitKBps, 0, 1<<20)
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
	Hidden    map[string]bool  `json:"hidden"` // fora do Início, consoles e busca
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
	if s.Hidden == nil {
		s.Hidden = map[string]bool{}
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
		Hidden:    maps.Clone(s.state.Hidden),
	}
}

// RecordPlay soma o tempo de jogo e marca como jogado agora.
func (s *Store) RecordPlay(id string, d time.Duration, now time.Time) error {
	return s.UpdateState(func(st *State) {
		st.LastPlay[id] = now.Unix()
		st.PlayTime[id] += int64(d.Seconds())
	})
}
