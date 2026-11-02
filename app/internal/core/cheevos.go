package core

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/stanleygomes/arkflix-games/app/internal/cheevos"
)

// cheevosTTL: quanto tempo as conquistas de um jogo ficam em memória.
const cheevosTTL = 10 * time.Minute

// cheevosState é o que o RetroAchievements tem de global: o cliente, as
// conquistas recentes por perfil+jogo e os hashes das ROMs (em disco: hash de
// ISO de PS2 lê megabytes).
type cheevosState struct {
	agent  string
	hashes string // <cache>/cheevos/hashes.json

	mu    sync.Mutex
	games map[string]cachedCheevos
	known map[string]cachedHash
}

type cachedCheevos struct {
	at   time.Time
	game GameCheevos
}

type cachedHash struct {
	Size int64  `json:"size"`
	Mod  int64  `json:"mod"`
	Hash string `json:"hash"`
}

func newCheevos(cache, version string) *cheevosState {
	c := &cheevosState{
		agent:  "arkflix-games/" + version,
		hashes: filepath.Join(cache, "cheevos", "hashes.json"),
		games:  map[string]cachedCheevos{},
		known:  map[string]cachedHash{},
	}
	if raw, err := os.ReadFile(c.hashes); err == nil {
		json.Unmarshal(raw, &c.known)
	}
	return c
}

// client: ARKFLIX_RA_BASE troca o servidor (RA falso nos testes de ponta a
// ponta). Lido a cada uso: no modo de teste o endereço só existe depois do New.
func (c *cheevosState) client() *cheevos.Client {
	return &cheevos.Client{Base: os.Getenv("ARKFLIX_RA_BASE"), UserAgent: c.agent}
}

// hash calcula (ou lembra, pelo tamanho+data do arquivo) o hash do RA.
func (c *cheevosState) hash(path, system string) (string, error) {
	st, err := os.Stat(path)
	if err != nil {
		return "", err
	}
	c.mu.Lock()
	k, ok := c.known[path]
	c.mu.Unlock()
	if ok && k.Size == st.Size() && k.Mod == st.ModTime().UnixNano() {
		return k.Hash, nil
	}
	h, err := cheevos.Hash(path, system)
	if err != nil {
		return "", err
	}
	c.mu.Lock()
	c.known[path] = cachedHash{Size: st.Size(), Mod: st.ModTime().UnixNano(), Hash: h}
	err = writePrivate(c.hashes, c.known)
	c.mu.Unlock()
	if err != nil {
		log.Printf("cache de hashes do RetroAchievements: %v", err)
	}
	return h, nil
}

func (c *cheevosState) forget() {
	c.mu.Lock()
	c.games = map[string]cachedCheevos{}
	c.mu.Unlock()
}

// cheevosConn é a conta do perfil: usuário e token, nunca a senha.
type cheevosConn struct {
	cheevos.Account
	Connected int64 `json:"connected"`
}

func (u *user) cheevosFile() string { return filepath.Join(u.dir, "cheevos.json") }
func (u *user) cheevosCfg() string  { return filepath.Join(u.dir, "retroarch-cheevos.cfg") }

func (u *user) cheevosConn() (cheevosConn, bool) {
	var c cheevosConn
	raw, err := os.ReadFile(u.cheevosFile())
	if err != nil || json.Unmarshal(raw, &c) != nil || c.Token == "" || c.User == "" {
		return cheevosConn{}, false
	}
	return c, true
}

// writeCheevosCfg: config que o RetroArch anexa pra entrar sozinho. Modo normal
// (hardcore desliga save state, e o app vive de save state).
func (u *user) writeCheevosCfg(c cheevosConn) error {
	cfg := fmt.Sprintf("cheevos_enable = \"true\"\ncheevos_username = %q\ncheevos_token = %q\ncheevos_password = \"\"\ncheevos_hardcore_mode_enable = \"false\"\n",
		cfgValue(c.User), cfgValue(c.Token))
	tmp := u.cheevosCfg() + ".tmp"
	if err := os.WriteFile(tmp, []byte(cfg), 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, u.cheevosCfg())
}

// cfgValue: o .cfg do RetroArch não tem escape; aspas e quebras viram nada.
func cfgValue(s string) string {
	return strings.Map(func(r rune) rune {
		if r == '"' || r == '\n' || r == '\r' || r == '\\' {
			return -1
		}
		return r
	}, s)
}

// cheevosAppend: a config do RetroAchievements, se o perfil tiver conta.
// Sem conta, não anexa nada (vale o que estiver na config do RetroArch).
func (u *user) cheevosAppend() string {
	c, ok := u.cheevosConn()
	if !ok {
		return ""
	}
	if err := u.writeCheevosCfg(c); err != nil {
		log.Printf("config do RetroAchievements: %v", err)
		return ""
	}
	return u.cheevosCfg()
}

// CheevosInfo é o estado da conta do RetroAchievements do perfil.
type CheevosInfo struct {
	Connected bool   `json:"connected"`
	User      string `json:"user"`
	Hardcore  bool   `json:"hardcore"` // sempre falso por enquanto
	Since     int64  `json:"since"`
}

// CheevosStatus diz se o perfil em uso tem conta do RetroAchievements.
func (s *Service) CheevosStatus() CheevosInfo {
	c, ok := s.cur().cheevosConn()
	if !ok {
		return CheevosInfo{}
	}
	return CheevosInfo{Connected: true, User: c.User, Since: c.Connected}
}

// CheevosLogin entra no RetroAchievements com usuário e senha do site. A
// senha vira o token do RA e é descartada; os jogos do RetroArch passam a
// entrar sozinhos com a conta deste perfil.
func (s *Service) CheevosLogin(user, pass string) (CheevosInfo, error) {
	if err := s.adult(); err != nil {
		return CheevosInfo{}, err
	}
	user = strings.TrimSpace(user)
	if user == "" || pass == "" {
		return CheevosInfo{}, errors.New("informe usuário e senha")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	acc, err := s.cheevos.client().Login(ctx, user, pass)
	if err != nil {
		return CheevosInfo{}, err
	}
	u := s.cur()
	c := cheevosConn{Account: acc, Connected: time.Now().Unix()}
	if err := writePrivate(u.cheevosFile(), c); err != nil {
		return CheevosInfo{}, err
	}
	if err := u.writeCheevosCfg(c); err != nil {
		return CheevosInfo{}, err
	}
	s.cheevos.forget()
	return s.CheevosStatus(), nil
}

// CheevosLogout esquece a conta do perfil (token e config do RetroArch).
func (s *Service) CheevosLogout() error {
	if err := s.adult(); err != nil {
		return err
	}
	u := s.cur()
	for _, p := range []string{u.cheevosFile(), u.cheevosCfg()} {
		if err := os.Remove(p); err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		}
	}
	s.cheevos.forget()
	return nil
}

// Cheevo é uma conquista com o estado do perfil.
type Cheevo struct {
	ID          int    `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Points      int    `json:"points"`
	BadgeURL    string `json:"badgeUrl"` // colorida se desbloqueada, cinza se não
	Unlocked    bool   `json:"unlocked"`
	Hardcore    bool   `json:"hardcore"` // desbloqueada no modo hardcore
}

// GameCheevos são as conquistas do jogo. Supported falso = sem conquistas
// pra mostrar (Reason diz por quê); não é erro.
type GameCheevos struct {
	Connected      bool     `json:"connected"`
	Supported      bool     `json:"supported"`
	Reason         string   `json:"reason,omitempty"`
	RAGameID       int      `json:"raGameId"`
	Title          string   `json:"title"`
	Total          int      `json:"total"`
	Unlocked       int      `json:"unlocked"`
	Points         int      `json:"points"`         // soma de todas
	PointsUnlocked int      `json:"pointsUnlocked"` // soma das desbloqueadas
	List           []Cheevo `json:"list"`
}

// GameCheevos busca as conquistas do jogo e o que o perfil já desbloqueou.
// Só leitura: não abre sessão nem conta como "jogando agora" no site.
func (s *Service) GameCheevos(id string) (GameCheevos, error) {
	e, sys, err := s.find(id)
	if err != nil {
		return GameCheevos{}, err
	}
	u := s.cur()
	conn, ok := u.cheevosConn()
	if !ok {
		return GameCheevos{List: []Cheevo{}, Reason: "entre na sua conta do RetroAchievements"}, nil
	}
	key := u.id + "|" + id
	s.cheevos.mu.Lock()
	hit, cached := s.cheevos.games[key]
	s.cheevos.mu.Unlock()
	if cached && time.Since(hit.at) < cheevosTTL {
		return hit.game, nil
	}
	out, err := s.fetchCheevos(e, sys.Slug, conn.Account)
	if err != nil {
		return GameCheevos{}, err
	}
	s.cheevos.mu.Lock()
	s.cheevos.games[key] = cachedCheevos{at: time.Now(), game: out}
	s.cheevos.mu.Unlock()
	return out, nil
}

func (s *Service) fetchCheevos(e entry, system string, acc cheevos.Account) (GameCheevos, error) {
	out := GameCheevos{Connected: true, List: []Cheevo{}}
	if e.remote {
		out.Reason = "baixe o jogo pra ver as conquistas"
		return out, nil
	}
	hash, err := s.cheevos.hash(e.game.Launch, system)
	if cheevos.IsUnsupported(err) {
		out.Reason = err.Error()
		return out, nil
	}
	if err != nil {
		return GameCheevos{}, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	c := s.cheevos.client()
	if out.RAGameID, err = c.GameID(ctx, hash); err != nil {
		return GameCheevos{}, err
	}
	if out.RAGameID == 0 {
		out.Reason = "o RetroAchievements não reconhece esta versão do jogo"
		return out, nil
	}
	g, err := c.Game(ctx, acc, out.RAGameID)
	if err != nil {
		return GameCheevos{}, err
	}
	soft, err := c.Unlocks(ctx, acc, out.RAGameID, false)
	if err != nil {
		return GameCheevos{}, err
	}
	hard, err := c.Unlocks(ctx, acc, out.RAGameID, true)
	if err != nil {
		return GameCheevos{}, err
	}
	out.Supported, out.Title = true, g.Title
	fillCheevos(&out, g.Achievements, soft, hard)
	return out, nil
}

func fillCheevos(out *GameCheevos, list []cheevos.Achievement, soft, hard []int) {
	got := map[int]bool{}
	for _, id := range soft {
		got[id] = true
	}
	hc := map[int]bool{}
	for _, id := range hard {
		got[id], hc[id] = true, true
	}
	for _, a := range list {
		c := Cheevo{ID: a.ID, Title: a.Title, Description: a.Description, Points: a.Points,
			BadgeURL: a.BadgeLocked, Unlocked: got[a.ID], Hardcore: hc[a.ID]}
		if c.Unlocked {
			c.BadgeURL = a.Badge
			out.Unlocked++
			out.PointsUnlocked += a.Points
		}
		out.Total++
		out.Points += a.Points
		out.List = append(out.List, c)
	}
}
