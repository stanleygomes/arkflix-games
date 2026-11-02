// Package cheevos fala com o RetroAchievements pela API "connect"
// (dorequest.php, a mesma do rcheevos/RetroArch) e calcula o hash das ROMs.
// Só leitura: login, id do jogo pelo hash, conquistas e o que o usuário já
// desbloqueou. Nada que abra sessão, dê ping ou conceda conquista.
package cheevos

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// DefaultBase é o servidor oficial.
const DefaultBase = "https://retroachievements.org"

// MediaBase serve as imagens das conquistas.
const MediaBase = "https://media.retroachievements.org"

// ErrAuth: usuário/senha ou token recusados.
var ErrAuth = errors.New("usuário ou senha do RetroAchievements incorretos")

// ErrToken: o token salvo não vale mais (senha trocada no site, por exemplo).
var ErrToken = errors.New("a sessão do RetroAchievements expirou: entre de novo")

// Client chama o servidor. Base e HTTP trocáveis pros testes.
type Client struct {
	Base      string
	HTTP      *http.Client
	UserAgent string
}

// Account é o que fica salvo: o usuário e o token (nunca a senha).
type Account struct {
	User  string `json:"user"`
	Token string `json:"token"`
}

// Achievement é uma conquista do conjunto principal do jogo.
type Achievement struct {
	ID          int    `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Points      int    `json:"points"`
	Badge       string `json:"badge"`       // imagem colorida
	BadgeLocked string `json:"badgeLocked"` // imagem cinza
}

// Game é o conjunto de conquistas de um jogo.
type Game struct {
	ID           int           `json:"id"`
	Title        string        `json:"title"`
	Achievements []Achievement `json:"achievements"`
}

// reply é o envelope comum das respostas.
type reply struct {
	Success bool   `json:"Success"`
	Error   string `json:"Error"`
	Code    string `json:"Code"`
}

func (c *Client) base() string {
	if c.Base == "" {
		return DefaultBase
	}
	return strings.TrimRight(c.Base, "/")
}

// call faz o POST form-urlencoded (como o rcheevos) e decodifica em out.
func (c *Client) call(ctx context.Context, form url.Values, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.base()+"/dorequest.php", strings.NewReader(form.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	ua := c.UserAgent
	if ua == "" {
		ua = "arkflix-games"
	}
	req.Header.Set("User-Agent", ua)
	hc := c.HTTP
	if hc == nil {
		hc = &http.Client{Timeout: 20 * time.Second}
	}
	resp, err := hc.Do(req)
	if err != nil {
		return fmt.Errorf("RetroAchievements fora do ar: %w", err)
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if err != nil {
		return err
	}
	var r reply
	if err := json.Unmarshal(raw, &r); err != nil {
		return fmt.Errorf("resposta inesperada do RetroAchievements (HTTP %d)", resp.StatusCode)
	}
	if !r.Success {
		if r.Code == "invalid_credentials" || resp.StatusCode == http.StatusUnauthorized {
			if form.Get("p") != "" {
				return ErrAuth
			}
			return ErrToken
		}
		if r.Error == "" {
			r.Error = fmt.Sprintf("HTTP %d", resp.StatusCode)
		}
		return fmt.Errorf("RetroAchievements: %s", r.Error)
	}
	return json.Unmarshal(raw, out)
}

// Login troca usuário e senha pelo token (r=login2).
func (c *Client) Login(ctx context.Context, user, pass string) (Account, error) {
	var out struct {
		User  string `json:"User"`
		Token string `json:"Token"`
	}
	if err := c.call(ctx, url.Values{"r": {"login2"}, "u": {user}, "p": {pass}}, &out); err != nil {
		return Account{}, err
	}
	if out.Token == "" {
		return Account{}, errors.New("RetroAchievements não devolveu o token")
	}
	if out.User == "" {
		out.User = user
	}
	return Account{User: out.User, Token: out.Token}, nil
}

// GameID resolve o hash (r=gameid). 0 = o RA não conhece esta versão da ROM.
func (c *Client) GameID(ctx context.Context, hash string) (int, error) {
	var out struct {
		GameID int `json:"GameID"`
	}
	err := c.call(ctx, url.Values{"r": {"gameid"}, "m": {hash}}, &out)
	return out.GameID, err
}

// promoted: categoria das conquistas oficiais (5 = não promovidas, ficam de fora).
const promoted = 3

// Game busca as conquistas (r=achievementsets, o que o rc_client usa hoje).
// Só o conjunto principal ("core"); subconjuntos ficam de fora.
func (c *Client) Game(ctx context.Context, acc Account, id int) (Game, error) {
	var out struct {
		GameID int    `json:"GameId"`
		Title  string `json:"Title"`
		Sets   []struct {
			Type         string `json:"Type"`
			Achievements []struct {
				ID             int    `json:"ID"`
				Title          string `json:"Title"`
				Description    string `json:"Description"`
				Flags          int    `json:"Flags"`
				Points         int    `json:"Points"`
				BadgeName      string `json:"BadgeName"`
				BadgeURL       string `json:"BadgeURL"`
				BadgeLockedURL string `json:"BadgeLockedURL"`
			} `json:"Achievements"`
		} `json:"Sets"`
	}
	form := url.Values{"r": {"achievementsets"}, "u": {acc.User}, "t": {acc.Token}, "g": {strconv.Itoa(id)}}
	if err := c.call(ctx, form, &out); err != nil {
		return Game{}, err
	}
	g := Game{ID: out.GameID, Title: out.Title, Achievements: []Achievement{}}
	for i, set := range out.Sets {
		if set.Type != "core" && !(set.Type == "" && i == 0) {
			continue
		}
		for _, a := range set.Achievements {
			if a.Flags != promoted {
				continue
			}
			if a.BadgeURL == "" {
				a.BadgeURL = MediaBase + "/Badge/" + a.BadgeName + ".png"
			}
			if a.BadgeLockedURL == "" {
				a.BadgeLockedURL = MediaBase + "/Badge/" + a.BadgeName + "_lock.png"
			}
			g.Achievements = append(g.Achievements, Achievement{
				ID: a.ID, Title: a.Title, Description: a.Description, Points: a.Points,
				Badge: a.BadgeURL, BadgeLocked: a.BadgeLockedURL,
			})
		}
	}
	return g, nil
}

// Unlocks: ids das conquistas que o usuário já tem (r=unlocks, só leitura;
// o startsession também devolve isso, mas marca o jogo como "jogando agora").
// O servidor não manda a data por aqui.
func (c *Client) Unlocks(ctx context.Context, acc Account, id int, hardcore bool) ([]int, error) {
	var out struct {
		UserUnlocks []int `json:"UserUnlocks"`
	}
	h := "0"
	if hardcore {
		h = "1"
	}
	form := url.Values{"r": {"unlocks"}, "u": {acc.User}, "t": {acc.Token}, "g": {strconv.Itoa(id)}, "h": {h}}
	err := c.call(ctx, form, &out)
	return out.UserUnlocks, err
}
