// Package core é o serviço que a interface chama. Os métodos exportados de
// Service viram a API: no desktop pelo Wails, no modo web por HTTP (mesmos
// nomes e argumentos). Eventos (controle, jogo rodando) saem pelo Emitter.
//
// Tudo que pode morar num servidor no futuro entra por interface: de onde vêm
// os jogos (GameSource), os metadados (MetaProvider) e os saves (SaveStore).
// Hoje as três são locais; trocar por remotas não mexe na interface nem na API.
package core

import (
	"context"
	"encoding/base64"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/stanleygomes/arkflix-games/app/internal/catalog"
	"github.com/stanleygomes/arkflix-games/app/internal/gamepad"
	"github.com/stanleygomes/arkflix-games/app/internal/history"
	"github.com/stanleygomes/arkflix-games/app/internal/hooks"
	"github.com/stanleygomes/arkflix-games/app/internal/importer"
	"github.com/stanleygomes/arkflix-games/app/internal/input"
	"github.com/stanleygomes/arkflix-games/app/internal/launcher"
	"github.com/stanleygomes/arkflix-games/app/internal/library"
	"github.com/stanleygomes/arkflix-games/app/internal/meta"
	"github.com/stanleygomes/arkflix-games/app/internal/saves"
	"github.com/stanleygomes/arkflix-games/app/internal/settings"
)

// GameSource lista os jogos disponíveis.
type GameSource interface {
	Scan() []library.Game
}

// MetaProvider dá nome, descrição, ano, gêneros e capa.
type MetaProvider interface {
	Lookup(g library.Game) meta.Info
	Refresh()
}

// SaveStore lista, apaga e mostra os saves.
type SaveStore interface {
	List(g saves.Game) saves.Listing
	Delete(path string) error
	Owns(path string) bool
	Thumbnail(path string) ([]byte, string, error)
}

// Hooks roda scripts do usuário nos eventos de jogo.
type Hooks interface {
	Run(event string, args ...string)
	Scripts() map[string][]string
}

// Emitter manda eventos pra interface.
type Emitter interface {
	Emit(name string, data any)
}

// Window é o que só existe no desktop (janela, diálogo de arquivos).
type Window interface {
	Hide()
	Show()
	PickFiles() ([]string, error)
	ToggleFullscreen()
	Quit()
}

type noWindow struct{}

func (noWindow) Hide()             {}
func (noWindow) Show()             {}
func (noWindow) ToggleFullscreen() {}
func (noWindow) Quit()             {}
func (noWindow) PickFiles() ([]string, error) {
	return nil, errors.New("seletor de arquivos só no app desktop")
}

type nopEmitter struct{}

func (nopEmitter) Emit(string, any) {}

// GameView é o jogo como a interface mostra.
type GameView struct {
	library.Game
	Name       string   `json:"name"`
	Summary    string   `json:"summary"`
	Year       int      `json:"year"`
	Genres     []string `json:"genres"`
	Players    string   `json:"players"`
	Cover      string   `json:"cover"`
	Favorite   bool     `json:"favorite"`
	LastPlayed int64    `json:"lastPlayed"`
	PlayTime   int64    `json:"playTime"`
	Plays      int      `json:"plays"`
}

// Options vem de quem inicia o app (main). Fonte, metadados e saves vazios
// usam as implementações locais.
type Options struct {
	Version string
	Mode    string // desktop ou web
	Store   *settings.Store
	Pads    *gamepad.Manager
	Emitter Emitter
	Window  Window
	Source  GameSource
	Meta    MetaProvider
	Saves   SaveStore
	Hooks   Hooks
	// MinPlay: duração mínima de uma partida (0 = 10 s; negativo = tudo conta, testes).
	MinPlay time.Duration
}

type entry struct {
	game  library.Game
	info  meta.Info
	cover string
}

// Service é a API do app.
type Service struct {
	opt     Options
	inputs  input.Store
	ready   chan struct{}
	readyMu sync.Once

	hist *history.Log

	mu      sync.Mutex
	entries []entry
	running string
	plays   map[string]int
}

// New prepara o serviço: gera a config de entrada do RetroArch, liga os
// eventos do controle e varre a biblioteca.
func New(opt Options) (*Service, error) {
	if opt.Window == nil {
		opt.Window = noWindow{}
	}
	if opt.Emitter == nil {
		opt.Emitter = nopEmitter{}
	}
	store := opt.Store
	if opt.Source == nil {
		opt.Source = library.LocalSource{Dirs: func() []settings.LibraryDir { return store.Settings().LibraryDirs }}
	}
	if opt.Meta == nil {
		opt.Meta = &meta.Local{ESDEDir: func() string { return store.Settings().ESDEDir }}
	}
	if opt.Saves == nil {
		opt.Saves = saves.Local{Dirs: func() settings.SaveDirs { return store.Settings().Saves }}
	}
	if opt.Hooks == nil {
		runner := &hooks.Runner{Dir: filepath.Join(store.Dir(), "scripts"), Log: filepath.Join(store.Dir(), "logs", "scripts.log")}
		if err := runner.Prepare(); err != nil {
			return nil, err
		}
		opt.Hooks = runner
	}
	s := &Service{opt: opt, inputs: input.Store{Dir: store.Dir()}, ready: make(chan struct{})}
	s.hist = &history.Log{Path: filepath.Join(store.Dir(), "sessions.jsonl"), MinPlay: opt.MinPlay}
	sessions, err := s.hist.All()
	if err != nil {
		return nil, err
	}
	s.plays = s.hist.Plays(sessions)
	m, err := s.inputs.Load()
	if err != nil {
		m = input.Default()
	}
	if err := s.inputs.WriteConfig(m); err != nil {
		return nil, err
	}
	if opt.Pads != nil {
		opt.Pads.Subscribe(func(e gamepad.Event) { s.opt.Emitter.Emit("pad", e) })
		opt.Pads.OnPadsChanged(func(p []gamepad.Pad) { s.opt.Emitter.Emit("pads", p) })
	}
	s.scan()
	return s, nil
}

func (s *Service) scan() {
	s.opt.Meta.Refresh()
	games := s.opt.Source.Scan()
	entries := make([]entry, len(games))
	for i, g := range games {
		entries[i] = entry{game: g, info: s.opt.Meta.Lookup(g)}
	}
	s.mu.Lock()
	s.entries = entries
	s.mu.Unlock()
}

func (s *Service) view(e entry, st settings.State) GameView {
	g := e.game
	v := GameView{
		Game: g, Name: g.Title, Summary: e.info.Summary, Year: e.info.Year, Genres: e.info.Genres,
		Players: e.info.Players, Favorite: st.Favorites[g.ID], LastPlayed: st.LastPlay[g.ID], PlayTime: st.PlayTime[g.ID],
		Plays: s.plays[g.ID],
	}
	if e.info.Name != "" {
		v.Name = e.info.Name
	}
	if v.Genres == nil {
		v.Genres = []string{}
	}
	if e.info.Cover != "" {
		v.Cover = "/media/cover/" + g.ID
	}
	return v
}

func (s *Service) views() []GameView {
	st := s.opt.Store.State()
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]GameView, len(s.entries))
	for i, e := range s.entries {
		out[i] = s.view(e, st)
	}
	return out
}

func (s *Service) find(id string) (entry, catalog.System, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, e := range s.entries {
		if e.game.ID == id {
			sys, _ := catalog.Find(e.game.System)
			return e, sys, nil
		}
	}
	return entry{}, catalog.System{}, errors.New("jogo não encontrado (a biblioteca mudou?)")
}

// Boot é tudo que a interface precisa pra abrir.
type Boot struct {
	Version  string            `json:"version"`
	Mode     string            `json:"mode"`
	Systems  []catalog.System  `json:"systems"`
	Games    []GameView        `json:"games"`
	Settings settings.Settings `json:"settings"`
	Input    input.Mapping     `json:"input"`
	RetroPad []input.Button    `json:"retroPad"`
	Pads     []gamepad.Pad     `json:"pads"`
	Running  string            `json:"running"`
}

// Bootstrap devolve o estado inicial.
func (s *Service) Bootstrap() (Boot, error) {
	m, _ := s.inputs.Load()
	b := Boot{
		Version: s.opt.Version, Mode: s.opt.Mode, Systems: catalog.Systems,
		Games: s.views(), Settings: s.opt.Store.Settings(), Input: m, RetroPad: input.RetroPad,
		Pads: s.Pads(), Running: s.Running(),
	}
	return b, nil
}

// Ready é chamado pela interface depois de desenhar a primeira tela.
func (s *Service) Ready() { s.readyMu.Do(func() { close(s.ready) }) }

// WaitReady espera a interface (teste de fumaça do desktop).
func (s *Service) WaitReady(timeout time.Duration) bool {
	select {
	case <-s.ready:
		return true
	case <-time.After(timeout):
		return false
	}
}

// Refresh varre a biblioteca e os metadados de novo.
func (s *Service) Refresh() []GameView {
	s.scan()
	return s.views()
}

// SetFavorite marca ou desmarca o favorito.
func (s *Service) SetFavorite(id string, fav bool) (GameView, error) {
	e, _, err := s.find(id)
	if err != nil {
		return GameView{}, err
	}
	err = s.opt.Store.UpdateState(func(st *settings.State) {
		if fav {
			st.Favorites[id] = true
		} else {
			delete(st.Favorites, id)
		}
	})
	if err != nil {
		return GameView{}, err
	}
	st := s.opt.Store.State()
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.view(e, st), nil
}

// GameEvent avisa a interface sobre o jogo em execução.
type GameEvent struct {
	State   string `json:"state"` // running ou exited
	ID      string `json:"id"`
	Name    string `json:"name"`
	Seconds int64  `json:"seconds,omitempty"`
	Code    int    `json:"code,omitempty"`
	Error   string `json:"error,omitempty"`
}

func encodeToken(path string) string { return base64.RawURLEncoding.EncodeToString([]byte(path)) }

func (s *Service) decodeToken(token string) (string, error) {
	raw, err := base64.RawURLEncoding.DecodeString(token)
	if err != nil || !s.opt.Saves.Owns(string(raw)) {
		return "", errors.New("save state inválido")
	}
	return string(raw), nil
}

// Play abre o jogo. stateID vazio = do início; "auto" = de onde parou; ou o id
// de um save state da lista. Volta na hora; o fim chega pelo evento "game".
func (s *Service) Play(id, stateID string) error {
	e, sys, err := s.find(id)
	if err != nil {
		return err
	}
	g := e.game
	var resume *launcher.Resume
	mode := "start"
	switch stateID {
	case "":
	case "auto":
		resume, mode = &launcher.Resume{Slot: -1}, "auto"
	default:
		mode = "slot"
		path, err := s.decodeToken(stateID)
		if err != nil {
			return err
		}
		for _, st := range s.opt.Saves.List(saves.Game{System: sys, Launch: g.Launch}).States {
			if st.Path == path {
				resume = &launcher.Resume{Slot: st.Slot, File: path}
			}
		}
		if resume == nil {
			return errors.New("save state não é deste jogo")
		}
		if sys.Emulator != catalog.RetroArch {
			resume.Slot = 0
		}
	}
	cfg := s.opt.Store.Settings()
	spec := launcher.Spec{Bin: cfg.EmulatorBin, Cores: cfg.CoresDir, Fullscreen: cfg.Fullscreen, RAAppend: []string{s.inputs.ConfigPath()}}
	args, env, err := launcher.Command(sys, g.Launch, spec, resume)
	if err != nil {
		return err
	}
	s.mu.Lock()
	if s.running != "" {
		s.mu.Unlock()
		return errors.New("já tem um jogo aberto")
	}
	s.running = id
	s.mu.Unlock()

	s.mu.Lock()
	name := s.view(e, settings.State{}).Name
	s.mu.Unlock()
	started := time.Now()
	s.opt.Emitter.Emit("game", GameEvent{State: "running", ID: id, Name: name})
	s.opt.Hooks.Run(hooks.GameStart, g.Launch, name, sys.Slug)
	s.opt.Window.Hide()
	go func() {
		res, err := launcher.Run(context.Background(), args, env)
		s.opt.Window.Show()
		s.opt.Store.RecordPlay(id, res.Duration, time.Now())
		session := history.Session{Game: id, System: sys.Slug, Name: name, Start: started.Unix(), Seconds: int64(res.Duration.Seconds()), Mode: mode, Code: res.Code}
		if err != nil {
			session.Code = -1
		}
		s.hist.Append(session)
		s.mu.Lock()
		s.running = ""
		if s.hist.Counts(session) {
			s.plays[id]++
		}
		s.mu.Unlock()
		ev := GameEvent{State: "exited", ID: id, Name: name, Seconds: int64(res.Duration.Seconds()), Code: res.Code}
		if err != nil {
			ev.Error = err.Error()
		} else if res.Code != 0 {
			ev.Error = "o emulador fechou com erro " + strconv.Itoa(res.Code)
			if res.Tail != "" {
				ev.Error += ":\n" + res.Tail
			}
		}
		s.opt.Emitter.Emit("game", ev)
		s.opt.Hooks.Run(hooks.GameEnd, g.Launch, name, sys.Slug, strconv.FormatInt(ev.Seconds, 10))
	}()
	return nil
}

// ScriptsInfo mostra onde ficam os scripts de evento e quais existem.
type ScriptsInfo struct {
	Dir    string              `json:"dir"`
	Log    string              `json:"log"`
	Events map[string][]string `json:"events"`
}

// Scripts lista os scripts de evento instalados.
func (s *Service) Scripts() ScriptsInfo {
	return ScriptsInfo{
		Dir:    filepath.Join(s.opt.Store.Dir(), "scripts"),
		Log:    filepath.Join(s.opt.Store.Dir(), "logs", "scripts.log"),
		Events: s.opt.Hooks.Scripts(),
	}
}

// Running diz qual jogo está aberto.
func (s *Service) Running() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.running
}

// Saves lista os saves do jogo.
func (s *Service) Saves(id string) (saves.Listing, error) {
	e, sys, err := s.find(id)
	if err != nil {
		return saves.Listing{}, err
	}
	l := s.opt.Saves.List(saves.Game{System: sys, Launch: e.game.Launch})
	for i := range l.States {
		l.States[i].Token = encodeToken(l.States[i].Path)
	}
	if l.States == nil {
		l.States = []saves.State{}
	}
	if l.Files == nil {
		l.Files = []saves.File{}
	}
	return l, nil
}

// DeleteState apaga um save state do jogo.
func (s *Service) DeleteState(id, stateID string) (saves.Listing, error) {
	path, err := s.decodeToken(stateID)
	if err != nil {
		return saves.Listing{}, err
	}
	if err := s.opt.Saves.Delete(path); err != nil {
		return saves.Listing{}, err
	}
	return s.Saves(id)
}

// AnalyzeImport inspeciona arquivos/pastas antes de importar.
func (s *Service) AnalyzeImport(paths []string) []importer.Item {
	var clean []string
	for _, p := range paths {
		if p = strings.TrimSpace(p); p != "" {
			if strings.HasPrefix(p, "~/") {
				home, _ := os.UserHomeDir()
				p = filepath.Join(home, p[2:])
			}
			clean = append(clean, p)
		}
	}
	items := importer.Analyze(clean)
	if items == nil {
		items = []importer.Item{}
	}
	return items
}

// Import copia os itens pra pasta de importação e atualiza a biblioteca.
func (s *Service) Import(items []importer.Item) ([]importer.Result, error) {
	dest := s.opt.Store.Settings().ImportDir
	if dest == "" {
		return nil, errors.New("pasta de importação não configurada")
	}
	res := importer.Import(items, dest)
	s.scan()
	s.opt.Emitter.Emit("library", s.views())
	return res, nil
}

// PickFiles abre o seletor de arquivos do sistema (só no desktop).
func (s *Service) PickFiles() ([]string, error) { return s.opt.Window.PickFiles() }

// ToggleFullscreen alterna tela cheia (desktop).
func (s *Service) ToggleFullscreen() { s.opt.Window.ToggleFullscreen() }

// Quit fecha o app (desktop).
func (s *Service) Quit() { s.opt.Window.Quit() }

// GetSettings devolve as configurações.
func (s *Service) GetSettings() settings.Settings { return s.opt.Store.Settings() }

// SaveSettings grava as configurações e varre a biblioteca de novo.
func (s *Service) SaveSettings(next settings.Settings) (settings.Settings, error) {
	saved, err := s.opt.Store.Save(next)
	if err != nil {
		return saved, err
	}
	s.scan()
	s.opt.Emitter.Emit("library", s.views())
	return saved, nil
}

// ResetSettings volta ao padrão.
func (s *Service) ResetSettings() (settings.Settings, error) {
	saved, err := s.opt.Store.Reset()
	s.scan()
	s.opt.Emitter.Emit("library", s.views())
	return saved, err
}

// InputResult é o mapeamento salvo e os conflitos encontrados.
type InputResult struct {
	Mapping   input.Mapping `json:"mapping"`
	Conflicts []string      `json:"conflicts"`
}

// GetInput devolve o mapeamento de controle e teclado.
func (s *Service) GetInput() (input.Mapping, error) { return s.inputs.Load() }

// SaveInput grava o mapeamento e regenera a config do RetroArch.
func (s *Service) SaveInput(m input.Mapping) (InputResult, error) {
	saved, err := s.inputs.Save(m)
	if err != nil {
		return InputResult{}, err
	}
	c := saved.Conflicts()
	if c == nil {
		c = []string{}
	}
	return InputResult{Mapping: saved, Conflicts: c}, nil
}

// ResetInput volta o mapeamento ao padrão.
func (s *Service) ResetInput() (InputResult, error) { return s.SaveInput(input.Default()) }

// Pads lista os controles conectados.
func (s *Service) Pads() []gamepad.Pad {
	if s.opt.Pads == nil {
		return []gamepad.Pad{}
	}
	return s.opt.Pads.Pads()
}

// Media é o handler das imagens. Fica fora do Service pra não virar método da
// API (o Wails expõe todo método exportado do Service pra interface).
func Media(s *Service) http.Handler { return http.HandlerFunc(s.serveMedia) }

// serveMedia entrega capas (/media/cover/<jogo>) e miniaturas de save state
// (/media/thumb/<id>).
func (s *Service) serveMedia(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/media/"), "/")
	if len(parts) != 2 {
		http.NotFound(w, r)
		return
	}
	switch parts[0] {
	case "thumb":
		path, err := s.decodeToken(parts[1])
		if err != nil {
			http.NotFound(w, r)
			return
		}
		data, ctype, err := s.opt.Saves.Thumbnail(path)
		if err != nil {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", ctype)
		w.Header().Set("Cache-Control", "no-cache")
		w.Write(data)
	case "cover":
		e, _, err := s.find(parts[1])
		if err != nil || e.info.Cover == "" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Cache-Control", "no-cache")
		http.ServeFile(w, r, e.info.Cover)
	default:
		http.NotFound(w, r)
	}
}
