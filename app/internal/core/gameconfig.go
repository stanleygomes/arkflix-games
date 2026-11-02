package core

import (
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"sync"

	"github.com/stanleygomes/arkflix-games/app/internal/catalog"
	"github.com/stanleygomes/arkflix-games/app/internal/launcher"
)

// GameSettings é a configuração de um jogo no RetroArch (vazio = padrão).
type GameSettings struct {
	Core   string `json:"core"`   // core alternativo instalado
	Image  string `json:"image"`  // sharp, smooth ou crt
	Aspect string `json:"aspect"` // 4:3 ou full
}

func (g GameSettings) empty() bool { return g == GameSettings{} }

// CoreOption é um core que roda o console e está instalado.
type CoreOption struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Default bool   `json:"default"`
}

// GameConfigInfo é o que a tela "Configurar este jogo" mostra.
type GameConfigInfo struct {
	Supported bool         `json:"supported"` // só jogos do RetroArch
	Settings  GameSettings `json:"settings"`
	Cores     []CoreOption `json:"cores"`
	CRT       bool         `json:"crt"` // achou um shader de CRT no RetroArch
}

var (
	images  = []string{"", "sharp", "smooth", "crt"}
	aspects = []string{"", "4:3", "full"}
)

// gameConfigMu protege o gameconfig.json do perfil (id do jogo -> configuração).
var gameConfigMu sync.Mutex

func (u *user) gameConfigPath() string { return filepath.Join(u.dir, "gameconfig.json") }

func (u *user) gameConfigs() map[string]GameSettings {
	out := map[string]GameSettings{}
	if raw, err := os.ReadFile(u.gameConfigPath()); err == nil {
		json.Unmarshal(raw, &out)
	}
	return out
}

func (u *user) gameConfig(id string) GameSettings {
	gameConfigMu.Lock()
	defer gameConfigMu.Unlock()
	return u.gameConfigs()[id]
}

// GameConfig devolve a configuração do jogo e as opções possíveis.
func (s *Service) GameConfig(id string) (GameConfigInfo, error) {
	e, sys, err := s.find(id)
	if err != nil {
		return GameConfigInfo{}, err
	}
	info := GameConfigInfo{Cores: []CoreOption{}}
	if sys.Emulator != catalog.RetroArch || e.remote {
		return info, nil
	}
	info.Supported = true
	info.Settings = s.cur().gameConfig(id)
	info.Cores = s.installedCores(sys)
	_, info.CRT = s.crtShader()
	return info, nil
}

// SetGameConfig grava a configuração do jogo (só pra este perfil). Vale na
// próxima vez que ele abrir.
func (s *Service) SetGameConfig(id string, c GameSettings) (GameConfigInfo, error) {
	if err := s.adult(); err != nil {
		return GameConfigInfo{}, err
	}
	e, sys, err := s.find(id)
	if err != nil {
		return GameConfigInfo{}, err
	}
	if sys.Emulator != catalog.RetroArch || e.remote {
		return GameConfigInfo{}, errors.New("só dá pra configurar jogos do RetroArch baixados")
	}
	if c.Core == sys.Core {
		c.Core = ""
	}
	if c.Core != "" && !slices.ContainsFunc(s.installedCores(sys), func(o CoreOption) bool { return o.ID == c.Core }) {
		return GameConfigInfo{}, errors.New("core não instalado pra este console")
	}
	if !slices.Contains(images, c.Image) || !slices.Contains(aspects, c.Aspect) {
		return GameConfigInfo{}, errors.New("opção desconhecida")
	}
	if _, ok := s.crtShader(); c.Image == "crt" && !ok {
		return GameConfigInfo{}, errors.New("nenhum shader de CRT encontrado no RetroArch")
	}
	u := s.cur()
	gameConfigMu.Lock()
	all := u.gameConfigs()
	if c.empty() {
		delete(all, id)
	} else {
		all[id] = c
	}
	err = writePrivate(u.gameConfigPath(), all)
	gameConfigMu.Unlock()
	if err != nil {
		return GameConfigInfo{}, err
	}
	return s.GameConfig(id)
}

// launchSpec: como abrir os emuladores agora (configurações + o que veio do main).
func (s *Service) launchSpec() launcher.Spec {
	cfg := s.opt.Store.Settings()
	spec := s.opt.Emulators
	if spec.Bin == "" {
		spec.Exe = map[string]string{catalog.RetroArch: cfg.Emulators.RetroArch, catalog.PPSSPP: cfg.Emulators.PPSSPP, catalog.PCSX2: cfg.Emulators.PCSX2}
		if cfg.Emulators.Cores != "" {
			spec.Cores = cfg.Emulators.Cores
		}
	}
	return spec
}

func (s *Service) installedCores(sys catalog.System) []CoreOption {
	spec := s.launchSpec()
	dir, goos := launcher.CoresDir(spec), spec.GOOS
	if goos == "" {
		goos = runtime.GOOS
	}
	out := []CoreOption{}
	for _, c := range sys.Cores() {
		if _, err := os.Stat(launcher.CorePath(dir, c, goos)); err == nil || c == sys.Core {
			name := catalog.CoreNames[c]
			if name == "" {
				name = c
			}
			out = append(out, CoreOption{ID: c, Name: name, Default: c == sys.Core})
		}
	}
	return out
}

// applyGameConfig põe a configuração do jogo no comando: core, shader e uma
// config extra do RetroArch (a última do --appendconfig, então ganha das outras).
func (s *Service) applyGameConfig(u *user, id string, spec *launcher.Spec) {
	c := u.gameConfig(id)
	if c.empty() {
		return
	}
	spec.Core = c.Core
	var body strings.Builder
	switch c.Image {
	case "sharp":
		body.WriteString("video_smooth = \"false\"\nvideo_scale_integer = \"true\"\n")
	case "smooth":
		body.WriteString("video_smooth = \"true\"\nvideo_scale_integer = \"false\"\n")
	case "crt":
		if shader, ok := s.crtShader(); ok {
			spec.Shader = shader
			body.WriteString("video_shader_enable = \"true\"\n")
		}
	}
	// índices do RetroArch 1.15+: 0 = 4:3, 24 = tela toda
	switch c.Aspect {
	case "4:3":
		body.WriteString("video_aspect_ratio_auto = \"false\"\naspect_ratio_index = \"0\"\n")
	case "full":
		body.WriteString("video_aspect_ratio_auto = \"false\"\naspect_ratio_index = \"24\"\n")
	}
	if body.Len() == 0 {
		return
	}
	path := filepath.Join(u.dir, "retroarch-game.cfg")
	if err := os.WriteFile(path, []byte(body.String()), 0o644); err == nil {
		spec.RAAppend = append(spec.RAAppend, path)
	}
}

// crtPreferred: os CRT mais leves primeiro (rodam em qualquer placa).
var crtPreferred = []string{"crt-easymode", "zfast-crt", "crt-lottes", "crt-geom", "crt-pi", "crt-royale"}

// crtShader acha um preset de CRT nas pastas de shaders do RetroArch.
// Devolve o caminho como o RetroArch enxerga (no Flatpak, /app/...).
func (s *Service) crtShader() (string, bool) {
	raDir := filepath.Dir(s.opt.Store.Settings().Saves.RetroArchSaves)
	// driver gl usa .glslp; vulkan/glcore/d3d usam .slangp
	ext := ".slangp"
	if data, err := os.ReadFile(filepath.Join(raDir, "retroarch.cfg")); err == nil &&
		strings.Contains(string(data), "video_driver = \"gl\"") {
		ext = ".glslp"
	}
	type place struct{ host, seen string }
	places := []place{{filepath.Join(raDir, "shaders"), ""}}
	if s.opt.Emulators.Bin == "" {
		home, _ := os.UserHomeDir()
		switch runtime.GOOS {
		case "linux":
			const inApp = "/app/share/libretro/shaders"
			for _, base := range []string{filepath.Join(home, ".local", "share", "flatpak"), "/var/lib/flatpak"} {
				places = append(places, place{filepath.Join(base, "app", launcher.Flatpaks[catalog.RetroArch], "current", "active", "files", "share", "libretro", "shaders"), inApp})
			}
		case "windows":
			if exe := s.opt.Store.Settings().Emulators.RetroArch; exe != "" {
				places = append(places, place{filepath.Join(filepath.Dir(exe), "shaders"), ""})
			}
		}
	}
	for _, p := range places {
		if found := findCRT(p.host, ext); found != "" {
			if p.seen == "" {
				return found, true
			}
			rel, _ := filepath.Rel(p.host, found)
			return filepath.ToSlash(filepath.Join(p.seen, rel)), true
		}
	}
	return "", false
}

func findCRT(dir, ext string) string {
	var hits []string
	filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			rel, _ := filepath.Rel(dir, path)
			if strings.Count(rel, string(filepath.Separator)) >= 3 || (path != dir && strings.HasPrefix(d.Name(), ".")) {
				return fs.SkipDir
			}
			return nil
		}
		name := strings.ToLower(d.Name())
		if strings.HasSuffix(name, ext) && strings.Contains(name, "crt") {
			hits = append(hits, path)
		}
		return nil
	})
	if len(hits) == 0 {
		return ""
	}
	slices.Sort(hits)
	for _, want := range crtPreferred {
		for _, h := range hits {
			if strings.TrimSuffix(strings.ToLower(filepath.Base(h)), ext) == want {
				return h
			}
		}
	}
	return hits[0]
}
