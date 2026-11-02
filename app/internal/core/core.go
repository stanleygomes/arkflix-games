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
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/stanleygomes/arkflix-games/app/internal/bt"
	"github.com/stanleygomes/arkflix-games/app/internal/catalog"
	"github.com/stanleygomes/arkflix-games/app/internal/emucfg"
	"github.com/stanleygomes/arkflix-games/app/internal/gamepad"
	"github.com/stanleygomes/arkflix-games/app/internal/history"
	"github.com/stanleygomes/arkflix-games/app/internal/hooks"
	"github.com/stanleygomes/arkflix-games/app/internal/importer"
	"github.com/stanleygomes/arkflix-games/app/internal/input"
	"github.com/stanleygomes/arkflix-games/app/internal/launcher"
	"github.com/stanleygomes/arkflix-games/app/internal/library"
	"github.com/stanleygomes/arkflix-games/app/internal/lists"
	"github.com/stanleygomes/arkflix-games/app/internal/meta"
	"github.com/stanleygomes/arkflix-games/app/internal/profiles"
	"github.com/stanleygomes/arkflix-games/app/internal/saves"
	"github.com/stanleygomes/arkflix-games/app/internal/settings"
	"github.com/stanleygomes/arkflix-games/app/internal/stats"
	"github.com/stanleygomes/arkflix-games/app/internal/thumbs"
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
	IsFullscreen() bool
	Minimize()
	Quit()
	OpenFile(path string) error
}

type noWindow struct{}

func (noWindow) Hide()             {}
func (noWindow) Show()             {}
func (noWindow) ToggleFullscreen() {}
func (noWindow) Quit()             {}
func (noWindow) Minimize()         {}
func (noWindow) IsFullscreen() bool {
	return false
}
func (noWindow) OpenFile(string) error {
	return errors.New("abrir no sistema só no app desktop")
}
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
	Snap       string   `json:"snap"`
	TitleShot  string   `json:"titleShot"`
	Manual     string   `json:"manual"`
	Video      string   `json:"video"`
	Favorite   bool     `json:"favorite"`
	Hidden     bool     `json:"hidden"` // oculto pelo perfil (só a página e a lista de ocultos mostram)
	LastPlayed int64    `json:"lastPlayed"`
	PlayTime   int64    `json:"playTime"`
	Plays      int      `json:"plays"`
	HasSaves   bool     `json:"hasSaves"`
	Developer  string   `json:"developer"`
	Rating     int      `json:"rating"`
	Completed  bool     `json:"completed"`
	Broken     bool     `json:"broken"`
	Edited     bool     `json:"edited"`
	KidsOK     bool     `json:"kidsOk"`
	Remote     bool     `json:"remote"` // só no servidor (baixar antes de jogar)
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
	// Cache: pasta de cache (capas baixadas, bancos do libretro).
	Cache string
	// Emulators: onde estão os emuladores. Vazio = padrão do sistema.
	Emulators launcher.Spec
	// BrowserPads: a interface lê os controles mesmo no Linux (testes do
	// caminho do Windows/macOS).
	BrowserPads bool
	// Bluetooth: pareamento de controles. Vazio = o do sistema (testes passam
	// o bt.Fake, que nunca toca no rádio de verdade).
	Bluetooth bt.Backend
	// Update: atualização automática. Zero = não olha sozinho (testes).
	Update UpdateOptions
}

type entry struct {
	game     library.Game
	auto     meta.Info // o que veio do ES-DE/libretro/arquivo
	info     meta.Info // auto + edição
	edit     meta.Override
	hasSaves bool
	romID    int  // jogo no servidor RomM (0 = não está lá)
	remote   bool // só no servidor: ainda não baixado
}

// Service é a API do app.
type Service struct {
	opt     Options
	ready   chan struct{}
	readyMu sync.Once

	over     *meta.Overrides
	thumbs   *thumbs.Store
	profiles *profiles.Store
	remote   *remoteState
	cheevos  *cheevosState

	umu    sync.RWMutex
	u      *user // perfil em uso
	chosen bool  // alguém já escolheu o perfil nesta execução

	mu      sync.Mutex
	entries []entry
	running string
	tasks   map[string]context.CancelFunc
	game    *gameRun // jogo aberto (fechar pelo guia)

	bts btState // busca de Bluetooth
	upd *updateState
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
	if opt.Bluetooth == nil {
		opt.Bluetooth = bt.New()
	}
	store := opt.Store
	if opt.Source == nil {
		opt.Source = library.LocalSource{Dirs: func() []settings.LibraryDir { return store.Settings().LibraryDirs }}
	}
	if opt.Cache == "" {
		opt.Cache = filepath.Join(store.Dir(), "cache")
	}
	covers := &thumbs.Store{Cache: opt.Cache}
	if opt.Meta == nil {
		local := &meta.Local{ESDEDir: func() string { return store.Settings().ESDEDir }}
		opt.Meta = &thumbs.Provider{Base: local, Store: covers}
	}
	if opt.Hooks == nil {
		runner := &hooks.Runner{Dir: filepath.Join(store.Dir(), "scripts"), Log: filepath.Join(store.Dir(), "logs", "scripts.log")}
		if err := runner.Prepare(); err != nil {
			return nil, err
		}
		opt.Hooks = runner
	}
	s := &Service{opt: opt, ready: make(chan struct{}), thumbs: covers, tasks: map[string]context.CancelFunc{}}
	if s.opt.Saves == nil {
		s.opt.Saves = saves.Local{Dirs: s.saveDirs}
	}
	s.upd = s.newUpdateState()
	s.over = &meta.Overrides{Path: filepath.Join(store.Dir(), "metadata.json")}
	s.profiles = &profiles.Store{Dir: store.Dir()}
	if _, err := s.profiles.All(); err != nil {
		return nil, err
	}
	u, err := s.loadUser(profiles.Main)
	if err != nil {
		return nil, err
	}
	s.u = u
	s.remote = &remoteState{file: filepath.Join(store.Dir(), "romm.json"), dir: filepath.Join(opt.Cache, "romm")}
	s.remote.load()
	s.cheevos = newCheevos(opt.Cache, opt.Version)
	if opt.Pads != nil {
		opt.Pads.Subscribe(func(e gamepad.Event) { s.opt.Emitter.Emit("pad", e) })
		opt.Pads.OnPadsChanged(func(p []gamepad.Pad) { s.opt.Emitter.Emit("pads", p) })
	}
	s.scan()
	if conn, _ := s.remote.snapshot(); conn != nil {
		s.startRemoteRefresh() // catálogo em cache já aparece; o novo chega depois
	}
	if opt.Update.Check {
		go s.checkLater()
	}
	return s, nil
}

func (s *Service) scan() {
	s.opt.Meta.Refresh()
	games := s.opt.Source.Scan()
	entries := make([]entry, len(games))
	for i, g := range games {
		auto := s.opt.Meta.Lookup(g)
		edit := s.over.Get(g.ID)
		entries[i] = entry{game: g, auto: auto, info: edit.Apply(auto), edit: edit, hasSaves: s.hasSaves(g)}
	}
	entries = s.mergeRemote(entries)
	s.mu.Lock()
	s.entries = entries
	s.mu.Unlock()
}

// hasSaves: save state ou save do próprio jogo (memory card compartilhado não conta).
func (s *Service) hasSaves(g library.Game) bool {
	sys, ok := catalog.Find(g.System)
	if !ok {
		return false
	}
	l := s.opt.Saves.List(saves.Game{System: sys, Launch: g.Launch})
	if len(l.States) > 0 {
		return true
	}
	for _, f := range l.Files {
		if !f.Shared {
			return true
		}
	}
	return false
}

func (s *Service) view(e entry, st settings.State) GameView {
	g := e.game
	v := GameView{
		Game: g, Name: g.Title, Summary: e.info.Summary, Year: e.info.Year, Genres: e.info.Genres,
		Players: e.info.Players, Favorite: st.Favorites[g.ID], Hidden: st.Hidden[g.ID], LastPlayed: st.LastPlay[g.ID], PlayTime: st.PlayTime[g.ID],
		Plays: s.cur().plays[g.ID], HasSaves: e.hasSaves, Developer: e.info.Developer,
		Rating: e.edit.Rating, Completed: e.edit.Completed, Broken: e.edit.Broken, Edited: !e.edit.Empty(),
		KidsOK: e.edit.KidsOK, Remote: e.remote,
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
	if e.info.Snap != "" {
		v.Snap = "/media/snap/" + g.ID
	}
	if e.info.TitleShot != "" {
		v.TitleShot = "/media/title/" + g.ID
	}
	if e.info.Manual != "" {
		v.Manual = "/media/manual/" + g.ID
	}
	if e.info.Video != "" {
		v.Video = "/media/video/" + g.ID
	}
	return v
}

func (s *Service) views() []GameView {
	u := s.cur()
	st := u.store.State()
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]GameView, 0, len(s.entries))
	for _, e := range s.entries {
		if u.allows(e) {
			out = append(out, s.view(e, st))
		}
	}
	return out
}

func (s *Service) find(id string) (entry, catalog.System, error) {
	u := s.cur()
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, e := range s.entries {
		if e.game.ID == id {
			if !u.allows(e) {
				return entry{}, catalog.System{}, errors.New("jogo não liberado no perfil infantil")
			}
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
	Lists    []lists.List      `json:"lists"`
	Profile  ProfileState      `json:"profile"`
	// NativePads: o Go lê os controles (Linux); senão, a interface lê pela
	// Gamepad API do WebView (Windows, macOS).
	NativePads bool   `json:"nativePads"`
	Platform   string `json:"platform"`
}

// Bootstrap devolve o estado inicial.
func (s *Service) Bootstrap() (Boot, error) {
	u := s.cur()
	m, _ := u.inputs.Load()
	b := Boot{
		Version: s.opt.Version, Mode: s.opt.Mode, Systems: catalog.Systems,
		Games: s.views(), Settings: s.GetSettings(), Input: m, RetroPad: input.RetroPad,
		Pads: s.Pads(), Running: s.Running(), NativePads: gamepad.Native && !s.opt.BrowserPads, Platform: runtime.GOOS,
	}
	var err error
	if b.Lists, err = u.lists.All(); err != nil {
		return b, err
	}
	b.Profile, err = s.Profiles()
	return b, err
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
	err = s.cur().store.UpdateState(func(st *settings.State) {
		if fav {
			st.Favorites[id] = true
		} else {
			delete(st.Favorites, id)
		}
	})
	if err != nil {
		return GameView{}, err
	}
	st := s.cur().store.State()
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.view(e, st), nil
}

// SetHidden oculta ou mostra o jogo na biblioteca do perfil (não apaga nada).
func (s *Service) SetHidden(id string, hidden bool) (GameView, error) {
	if err := s.adult(); err != nil {
		return GameView{}, err
	}
	e, _, err := s.find(id)
	if err != nil {
		return GameView{}, err
	}
	err = s.cur().store.UpdateState(func(st *settings.State) {
		if hidden {
			st.Hidden[id] = true
		} else {
			delete(st.Hidden, id)
		}
	})
	if err != nil {
		return GameView{}, err
	}
	st := s.cur().store.State()
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.view(e, st), nil
}

// hideDelay: quanto o app espera o emulador abrir antes de esconder a janela.
const hideDelay = 1500 * time.Millisecond

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
	if e.remote {
		return errors.New("este jogo está no servidor: baixe antes de jogar")
	}
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
	u := s.cur()
	spec := s.launchSpec()
	if spec.Bin == "" {
		prepareEmulator(sys.Emulator, s.saveDirs())
	}
	spec.Fullscreen, spec.RAAppend, spec.Work = cfg.Fullscreen, append(u.raAppend(), s.guideCfg(u)...), u.dir
	if sys.Emulator == catalog.RetroArch {
		s.applyGameConfig(u, id, &spec)
	}
	args, err := launcher.Command(sys, g.Launch, spec, resume)
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

	if s.syncs(e, sys, u) {
		// saves mais novos de outro aparelho; sem servidor, joga com o daqui
		ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
		n, conflicts, err := s.pullSaves(ctx, e, sys, s.saveDirs(), u.lastSync(id))
		switch {
		case err != nil:
			s.opt.Emitter.Emit("task", TaskEvent{ID: "pull:" + id, Label: "Saves do servidor", State: "error", Message: err.Error()})
		case len(conflicts) > 0:
			s.opt.Emitter.Emit("task", TaskEvent{ID: "pull:" + id, Label: "Conflito de save", State: "error",
				Message: fmt.Sprintf("o save daqui e o do servidor mudaram. Fiquei com o daqui; o do servidor foi guardado como %s.servidor", conflicts[0])})
		case n > 0:
			s.opt.Emitter.Emit("task", TaskEvent{ID: "pull:" + id, Label: "Saves do servidor", State: "done", Progress: 1, Message: fmt.Sprintf("%d save(s) mais novo(s) baixado(s)", n)})
		}
		if err == nil && len(conflicts) == 0 {
			u.setSync(id, time.Now().Unix())
		}
		cancel()
	}

	s.mu.Lock()
	name := s.view(e, settings.State{}).Name
	s.mu.Unlock()
	started := time.Now()
	s.opt.Emitter.Emit("game", GameEvent{State: "running", ID: id, Name: name})
	s.opt.Hooks.Run(hooks.GameStart, g.Launch, name, sys.Slug)
	// A janela só some depois que o emulador teve tempo de abrir: enquanto
	// isso a interface mostra "PLAYER 1 START" e a área de trabalho não pisca.
	// O jogo fechou antes? Não esconde (e o mutex impede esconder depois do Show).
	var winMu sync.Mutex
	exited := false
	go func() {
		time.Sleep(hideDelay)
		winMu.Lock()
		if !exited {
			s.opt.Window.Hide()
		}
		winMu.Unlock()
	}()
	gctx, stop := context.WithCancel(context.Background())
	s.mu.Lock()
	s.game = &gameRun{id: id, emulator: sys.Emulator, stop: stop}
	s.mu.Unlock()
	go func() {
		defer stop()
		res, err := launcher.Run(gctx, args, nil)
		if gctx.Err() != nil {
			err, res.Code, res.Tail = nil, 0, "" // fechado pelo guia: não é erro
		}
		winMu.Lock()
		exited = true
		winMu.Unlock()
		s.opt.Window.Show()
		u.store.RecordPlay(id, res.Duration, time.Now())
		session := history.Session{Game: id, System: sys.Slug, Name: name, Start: started.Unix(), Seconds: int64(res.Duration.Seconds()), Mode: mode, Code: res.Code}
		if err != nil {
			session.Code = -1
		}
		u.hist.Append(session)
		s.mu.Lock()
		s.running, s.game = "", nil
		if u.hist.Counts(session) {
			u.plays[id]++
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
		// o script dispara antes do aviso: quem espera o "exited" já o vê rodando
		s.opt.Hooks.Run(hooks.GameEnd, g.Launch, name, sys.Slug, strconv.FormatInt(ev.Seconds, 10))
		if s.syncs(e, sys, u) && err == nil {
			listing := s.opt.Saves.List(saves.Game{System: sys, Launch: g.Launch})
			s.runTask("sync:"+id, "Enviando saves pro servidor", func(func(float64)) (string, error) {
				ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
				defer cancel()
				n, err := s.pushSaves(ctx, e, sys, listing, started.Unix()-2)
				if err == nil {
					u.setSync(id, time.Now().Unix())
				}
				return fmt.Sprintf("%d save(s) enviado(s)", n), err
			})
		}
		s.opt.Emitter.Emit("game", ev)
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
	if e.remote {
		return saves.Listing{States: []saves.State{}, Files: []saves.File{}, Note: "Baixe o jogo pra jogar e ver os saves."}, nil
	}
	l := s.opt.Saves.List(saves.Game{System: sys, Launch: e.game.Launch})
	if sys.Emulator != catalog.RetroArch && s.cur().id != profiles.Main && l.Note == "" {
		l.Note = "Saves de PSP e PS2 são compartilhados entre os perfis."
	}
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
	if err := s.adult(); err != nil {
		return saves.Listing{}, err
	}
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
	if s.adult() != nil {
		return []importer.Item{}
	}
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
	if err := s.adult(); err != nil {
		return nil, err
	}
	dest := s.opt.Store.Settings().ImportDir
	if dest == "" {
		return nil, errors.New("pasta de importação não configurada")
	}
	res := importer.Import(items, dest)
	s.scan()
	s.opt.Emitter.Emit("library", s.views())
	if s.opt.Store.Settings().Covers.Enabled {
		s.startCovers(false)
	}
	return res, nil
}

// PickFiles abre o seletor de arquivos do sistema (só no desktop).
func (s *Service) PickFiles() ([]string, error) {
	if err := s.adult(); err != nil {
		return nil, err
	}
	return s.opt.Window.PickFiles()
}

// ToggleFullscreen alterna tela cheia (desktop).
func (s *Service) ToggleFullscreen() { s.opt.Window.ToggleFullscreen() }

// IsFullscreen diz se a janela está em tela cheia (desktop).
func (s *Service) IsFullscreen() bool { return s.opt.Window.IsFullscreen() }

// Minimize minimiza a janela (desktop).
func (s *Service) Minimize() { s.opt.Window.Minimize() }

// Quit fecha o app (desktop).
func (s *Service) Quit() { s.opt.Window.Quit() }

// GetSettings devolve as configurações.
// No perfil que não é o principal, a acessibilidade vem do perfil.
func (s *Service) GetSettings() settings.Settings {
	g := s.opt.Store.Settings()
	if u := s.cur(); u.id != profiles.Main {
		g.A11y = u.store.Settings().A11y
	}
	return g
}

// SaveSettings grava as configurações e varre a biblioteca de novo.
func (s *Service) SaveSettings(next settings.Settings) (settings.Settings, error) {
	if err := s.adult(); err != nil {
		return s.GetSettings(), err
	}
	if u := s.cur(); u.id != profiles.Main {
		mine := u.store.Settings()
		mine.A11y = next.A11y
		if _, err := u.store.Save(mine); err != nil {
			return s.GetSettings(), err
		}
		next.A11y = s.opt.Store.Settings().A11y
	}
	if _, err := s.opt.Store.Save(next); err != nil {
		return s.GetSettings(), err
	}
	s.scan()
	s.opt.Emitter.Emit("library", s.views())
	return s.GetSettings(), nil
}

// ResetSettings volta ao padrão.
func (s *Service) ResetSettings() (settings.Settings, error) {
	if err := s.adult(); err != nil {
		return s.GetSettings(), err
	}
	_, err := s.opt.Store.Reset()
	if u := s.cur(); u.id != profiles.Main && err == nil {
		_, err = u.store.Reset()
	}
	saved := s.GetSettings()
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
func (s *Service) GetInput() (input.Mapping, error) { return s.cur().inputs.Load() }

// SaveInput grava o mapeamento e regenera a config do RetroArch.
func (s *Service) SaveInput(m input.Mapping) (InputResult, error) {
	if err := s.adult(); err != nil {
		return InputResult{}, err
	}
	saved, err := s.cur().inputs.Save(m)
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
	case "cover", "snap", "title", "manual", "video":
		e, _, err := s.find(parts[1])
		path := map[string]string{"cover": e.info.Cover, "snap": e.info.Snap, "title": e.info.TitleShot, "manual": e.info.Manual, "video": e.info.Video}[parts[0]]
		if err != nil || path == "" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Cache-Control", "no-cache")
		http.ServeFile(w, r, path)
	default:
		http.NotFound(w, r)
	}
}

// TaskEvent mostra o progresso de uma tarefa longa (capas).
type TaskEvent struct {
	ID       string  `json:"id"`
	Label    string  `json:"label"`
	State    string  `json:"state"` // running, done, error, cancelled
	Progress float64 `json:"progress"`
	Message  string  `json:"message,omitempty"`
}

// runTask roda fn em segundo plano, uma de cada id por vez, avisando o progresso.
func (s *Service) runTask(id, label string, fn func(progress func(float64)) (string, error)) bool {
	return s.runTaskCtx(id, label, func(_ context.Context, progress func(float64)) (string, error) { return fn(progress) })
}

// runTaskCtx é o runTask cancelável: CancelTask(id) cancela o ctx e o fim
// chega como "cancelled".
func (s *Service) runTaskCtx(id, label string, fn func(ctx context.Context, progress func(float64)) (string, error)) bool {
	s.mu.Lock()
	if _, busy := s.tasks[id]; busy {
		s.mu.Unlock()
		return false
	}
	ctx, cancel := context.WithCancel(context.Background())
	s.tasks[id] = cancel
	s.mu.Unlock()
	s.opt.Emitter.Emit("task", TaskEvent{ID: id, Label: label, State: "running"})
	go func() {
		defer cancel()
		msg, err := fn(ctx, func(p float64) {
			s.opt.Emitter.Emit("task", TaskEvent{ID: id, Label: label, State: "running", Progress: p})
		})
		s.mu.Lock()
		delete(s.tasks, id)
		s.mu.Unlock()
		ev := TaskEvent{ID: id, Label: label, State: "done", Progress: 1, Message: msg}
		switch {
		case err != nil && ctx.Err() != nil:
			ev.State, ev.Message, ev.Progress = "cancelled", "cancelado", 0
		case err != nil:
			ev.State, ev.Message = "error", err.Error()
		}
		s.opt.Emitter.Emit("task", ev)
	}()
	return true
}

// CancelTask cancela uma tarefa em andamento (download, por exemplo).
func (s *Service) CancelTask(id string) error {
	s.mu.Lock()
	cancel, ok := s.tasks[id]
	s.mu.Unlock()
	if !ok {
		return errors.New("essa tarefa já terminou")
	}
	cancel()
	return nil
}

// startCovers baixa as capas dos jogos que ainda não passaram pelo libretro
// (all = todos de novo).
func (s *Service) startCovers(all bool) bool {
	cfg := s.opt.Store.Settings().Covers
	s.thumbs.Configure(cfg.Images, cfg.DB)
	var todo []library.Game
	s.mu.Lock()
	for _, e := range s.entries {
		if !e.remote && thumbs.Supported(e.game.System) && (all || !s.thumbs.Fetched(e.game)) {
			todo = append(todo, e.game)
		}
	}
	s.mu.Unlock()
	return s.runTask("covers", "Baixando capas", func(progress func(float64)) (string, error) {
		found, missing := 0, 0
		defer func() {
			s.scan()
			s.opt.Emitter.Emit("library", s.views())
		}()
		for i, g := range todo {
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
			m, n, err := s.thumbs.Fetch(ctx, g)
			cancel()
			if err != nil {
				return "", err
			}
			if m.Name != "" && n > 0 {
				found++
			} else {
				missing++
			}
			progress(float64(i+1) / float64(len(todo)))
		}
		if len(todo) == 0 {
			return "todas as capas já estão em dia", nil
		}
		msg := fmt.Sprintf("%d jogo(s) com capa", found)
		if missing > 0 {
			msg += fmt.Sprintf(", %d sem capa no acervo do libretro", missing)
		}
		return msg, nil
	})
}

// UpdateCovers baixa capas, prints e telas de título do libretro
// (thumbnails.libretro.com). all = tenta de novo até as que não acharam.
func (s *Service) UpdateCovers(all bool) error {
	if err := s.adult(); err != nil {
		return err
	}
	if !s.opt.Store.Settings().Covers.Enabled {
		return errors.New("ligue \"Baixar capas\" nas configurações da biblioteca")
	}
	if !s.startCovers(all) {
		return errors.New("as capas já estão sendo baixadas")
	}
	return nil
}

// Lists devolve as listas pessoais.
func (s *Service) Lists() ([]lists.List, error) { return s.cur().lists.All() }

// CreateList cria uma lista vazia.
func (s *Service) CreateList(name string) (lists.List, error) { return s.cur().lists.Create(name) }

// RenameList troca o nome da lista.
func (s *Service) RenameList(id, name string) (lists.List, error) {
	return s.cur().lists.Rename(id, name)
}

// DeleteList apaga a lista (os jogos continuam na biblioteca).
func (s *Service) DeleteList(id string) error { return s.cur().lists.Delete(id) }

// SetInList põe ou tira o jogo da lista.
func (s *Service) SetInList(id, gameID string, in bool) (lists.List, error) {
	if _, _, err := s.find(gameID); err != nil {
		return lists.List{}, err
	}
	return s.cur().lists.Set(id, gameID, in)
}

// MetaInfo é o que o editor mostra: o automático e a edição.
type MetaInfo struct {
	Auto      meta.Info     `json:"auto"`
	AutoCover bool          `json:"autoCover"`
	Edit      meta.Override `json:"edit"`
}

// MetaDetails devolve os metadados automáticos e a edição do jogo.
func (s *Service) MetaDetails(id string) (MetaInfo, error) {
	e, _, err := s.find(id)
	if err != nil {
		return MetaInfo{}, err
	}
	auto := e.auto
	if auto.Name == "" {
		auto.Name = e.game.Title
	}
	if auto.Genres == nil {
		auto.Genres = []string{}
	}
	return MetaInfo{Auto: auto, AutoCover: e.auto.Cover != "", Edit: e.edit}, nil
}

// SaveMeta grava a edição do jogo e devolve o jogo atualizado.
func (s *Service) SaveMeta(id string, edit meta.Override) (GameView, error) {
	if err := s.adult(); err != nil {
		return GameView{}, err
	}
	if _, _, err := s.find(id); err != nil {
		return GameView{}, err
	}
	saved, err := s.over.Set(id, edit)
	if err != nil {
		return GameView{}, err
	}
	st := s.cur().store.State()
	s.mu.Lock()
	defer s.mu.Unlock()
	for i, e := range s.entries {
		if e.game.ID == id {
			s.entries[i].edit = saved
			s.entries[i].info = saved.Apply(e.auto)
			return s.view(s.entries[i], st), nil
		}
	}
	return GameView{}, errors.New("jogo não encontrado (a biblioteca mudou?)")
}

// ResetMeta volta tudo do jogo pro automático.
// A liberação pro perfil infantil não é metadado: continua.
func (s *Service) ResetMeta(id string) (GameView, error) {
	e, _, err := s.find(id)
	if err != nil {
		return GameView{}, err
	}
	return s.SaveMeta(id, meta.Override{KidsOK: e.edit.KidsOK})
}

// Stats resume o histórico de jogo no período ("7d", "30d" ou "all").
func (s *Service) Stats(period string) (stats.Stats, error) {
	if period != "7d" && period != "30d" && period != "all" {
		return stats.Stats{}, errors.New("período inválido")
	}
	u := s.cur()
	sessions, err := u.hist.All()
	if err != nil {
		return stats.Stats{}, err
	}
	var games []stats.Game
	for _, v := range s.views() {
		games = append(games, stats.Game{ID: v.ID, Name: v.Name, System: v.System})
	}
	return stats.Compute(u.hist, sessions, games, period, time.Now(), time.Local), nil
}

// OpenManual abre o manual do jogo no leitor de PDF do sistema (desktop).
func (s *Service) OpenManual(id string) error {
	e, _, err := s.find(id)
	if err != nil {
		return err
	}
	if e.info.Manual == "" {
		return errors.New("este jogo não tem manual")
	}
	return s.opt.Window.OpenFile(e.info.Manual)
}

// prepareEmulator deixa menu e saída iguais aos do RetroArch no PPSSPP e no
// PCSX2 (a config deles fica ao lado das pastas de save, em todo sistema).
// Falhar aqui não impede o jogo: abre com o padrão dele.
func prepareEmulator(emulator string, dirs settings.SaveDirs) {
	var err error
	switch emulator {
	case catalog.PPSSPP:
		err = emucfg.PPSSPP(filepath.Join(filepath.Dir(dirs.PPSSPPSaveData), "SYSTEM"))
	case catalog.PCSX2:
		err = emucfg.PCSX2(filepath.Join(filepath.Dir(dirs.PCSX2Memcards), "inis", "PCSX2.ini"))
	}
	if err != nil && !errors.Is(err, emucfg.ErrNotSetUp) {
		log.Printf("atalhos do %s: %v", emulator, err)
	}
}
