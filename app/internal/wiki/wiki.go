// Package wiki busca descrição, ano e gêneros de um jogo em fontes abertas e
// sem chave: o Wikidata (wbsearchentities + wbgetentities: instância de
// "video game", data de lançamento, gênero e plataforma pra desempatar pelo
// console) e o resumo da Wikipedia em português (inglês de reserva) pela API
// REST /page/summary. Só leitura, uma requisição por vez, com intervalo
// mínimo entre elas e User-Agent descritivo (regra da Wikimedia).
package wiki

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode"
	"unicode/utf8"
)

// ErrNotFound: nenhum jogo com esse nome no Wikidata.
var ErrNotFound = errors.New("não achei esse jogo no Wikidata")

// ErrBusy: a Wikimedia pediu pra esperar (429/503).
var ErrBusy = errors.New("a Wikipedia pediu pra esperar um pouco: tente de novo mais tarde")

// videoGame é o item "video game" (Q7889): a instância que conta.
const videoGame = "Q7889"

// Platforms liga o console do app aos itens de plataforma do Wikidata (P400).
var Platforms = map[string][]string{
	"nes":     {"Q172742"},
	"snes":    {"Q183259"},
	"genesis": {"Q10676"},
	"gbc":     {"Q203992"},
	"n64":     {"Q184839"},
	"psx":     {"Q10677"},
	"ps2":     {"Q10680"},
	"nds":     {"Q170323"},
	"psp":     {"Q170325"},
	"arcade":  {"Q192851", "Q631229"},
}

// Gate espaça as requisições (compartilhado entre buscas).
type Gate struct {
	Gap  time.Duration // 0 = 250 ms
	mu   sync.Mutex
	last time.Time
}

func (g *Gate) wait(ctx context.Context) error {
	if g == nil {
		return nil
	}
	gap := g.Gap
	if gap == 0 {
		gap = 250 * time.Millisecond
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	if d := time.Until(g.last.Add(gap)); d > 0 {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(d):
		}
	}
	g.last = time.Now()
	return nil
}

// Client chama o Wikidata e a Wikipedia. Base vazia = servidores oficiais;
// com Base (servidor falso dos testes), o Wikidata fica em Base/w/api.php e
// a Wikipedia em Base/<língua>/api/rest_v1/page/summary/.
type Client struct {
	Base      string
	HTTP      *http.Client
	UserAgent string
	Gate      *Gate
}

// Result é o que achou. Campos vazios = a fonte não tinha.
type Result struct {
	Entity  string   `json:"entity"` // Q... do Wikidata
	Title   string   `json:"title"`
	Summary string   `json:"summary"`
	Year    int      `json:"year"`
	Genres  []string `json:"genres"`
	Source  string   `json:"source"` // página de onde veio (Wikipedia ou Wikidata)
}

func (c *Client) wikidata() string {
	if c.Base != "" {
		return strings.TrimRight(c.Base, "/") + "/w/api.php"
	}
	return "https://www.wikidata.org/w/api.php"
}

func (c *Client) summaryURL(lang, title string) string {
	path := url.PathEscape(strings.ReplaceAll(title, " ", "_"))
	if c.Base != "" {
		return strings.TrimRight(c.Base, "/") + "/" + lang + "/api/rest_v1/page/summary/" + path
	}
	return "https://" + lang + ".wikipedia.org/api/rest_v1/page/summary/" + path
}

func (c *Client) get(ctx context.Context, u string, out any) (int, error) {
	if err := c.Gate.wait(ctx); err != nil {
		return 0, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return 0, err
	}
	ua := c.UserAgent
	if ua == "" {
		ua = "arkflix-games (https://github.com/stanleygomes/arkflix-games)"
	}
	req.Header.Set("User-Agent", ua)
	req.Header.Set("Accept", "application/json")
	hc := c.HTTP
	if hc == nil {
		hc = &http.Client{Timeout: 20 * time.Second}
	}
	resp, err := hc.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	switch {
	case resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode == http.StatusServiceUnavailable:
		return resp.StatusCode, ErrBusy
	case resp.StatusCode == http.StatusNotFound:
		return resp.StatusCode, nil
	case resp.StatusCode != http.StatusOK:
		return resp.StatusCode, fmt.Errorf("wikimedia respondeu %d", resp.StatusCode)
	}
	return resp.StatusCode, json.NewDecoder(io.LimitReader(resp.Body, 4<<20)).Decode(out)
}

func (c *Client) api(ctx context.Context, q url.Values, out any) error {
	q.Set("format", "json")
	_, err := c.get(ctx, c.wikidata()+"?"+q.Encode(), out)
	return err
}

type claim struct {
	Mainsnak struct {
		Datavalue struct {
			Value json.RawMessage `json:"value"`
		} `json:"datavalue"`
	} `json:"mainsnak"`
}

type entity struct {
	ID        string                            `json:"id"`
	Claims    map[string][]claim                `json:"claims"`
	Labels    map[string]struct{ Value string } `json:"labels"`
	Sitelinks map[string]struct{ Title string } `json:"sitelinks"`
}

func (e entity) items(prop string) []string {
	var out []string
	for _, c := range e.Claims[prop] {
		var v struct {
			ID string `json:"id"`
		}
		if json.Unmarshal(c.Mainsnak.Datavalue.Value, &v) == nil && v.ID != "" {
			out = append(out, v.ID)
		}
	}
	return out
}

// year: o mais antigo dos lançamentos (P577 tem um por região).
func (e entity) year() int {
	best := 0
	for _, c := range e.Claims["P577"] {
		var v struct {
			Time string `json:"time"`
		}
		if json.Unmarshal(c.Mainsnak.Datavalue.Value, &v) != nil || len(v.Time) < 5 {
			continue
		}
		y, err := strconv.Atoi(strings.SplitN(strings.TrimLeft(v.Time, "+"), "-", 2)[0])
		if err == nil && y >= 1950 && y <= 2100 && (best == 0 || y < best) {
			best = y
		}
	}
	return best
}

func (e entity) label() string {
	for _, l := range []string{"pt", "pt-br", "en"} {
		if v := e.Labels[l].Value; v != "" {
			return v
		}
	}
	return ""
}

func (c *Client) entities(ctx context.Context, ids []string, props string) (map[string]entity, error) {
	var out struct {
		Entities map[string]entity `json:"entities"`
	}
	q := url.Values{"action": {"wbgetentities"}, "ids": {strings.Join(ids, "|")}, "props": {props}, "languages": {"pt|pt-br|en"}}
	if strings.Contains(props, "sitelinks") {
		q.Set("sitefilter", "ptwiki|enwiki")
	}
	if err := c.api(ctx, q, &out); err != nil {
		return nil, err
	}
	return out.Entities, nil
}

var (
	noise  = regexp.MustCompile(`\s*[\(\[][^\)\]]*[\)\]]`)
	spaces = regexp.MustCompile(` {2,}`)
)

// norm: minúsculas, sem acento nem pontuação (comparar nomes).
func norm(s string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(s) {
		switch {
		case unicode.IsLetter(r) || unicode.IsDigit(r):
			b.WriteRune(r)
		case unicode.IsSpace(r) || r == '-' || r == ':':
			b.WriteByte(' ')
		}
	}
	return strings.Join(strings.Fields(b.String()), " ")
}

// Lookup acha o jogo pelo nome; system (snes, psx...) desempata pela plataforma.
func (c *Client) Lookup(ctx context.Context, name, system string) (Result, error) {
	name = strings.TrimSpace(noise.ReplaceAllString(name, ""))
	if name == "" {
		return Result{}, ErrNotFound
	}
	var search struct {
		Search []struct {
			ID    string `json:"id"`
			Label string `json:"label"`
		} `json:"search"`
	}
	q := url.Values{"action": {"wbsearchentities"}, "search": {name}, "language": {"en"}, "uselang": {"en"}, "type": {"item"}, "limit": {"10"}}
	if err := c.api(ctx, q, &search); err != nil {
		return Result{}, err
	}
	if len(search.Search) == 0 {
		return Result{}, ErrNotFound
	}
	var ids []string
	for _, s := range search.Search {
		ids = append(ids, s.ID)
	}
	ents, err := c.entities(ctx, ids, "claims|sitelinks|labels")
	if err != nil {
		return Result{}, err
	}
	best, bestScore := entity{}, -1
	want := Platforms[system]
	for i, id := range ids {
		e, ok := ents[id]
		if !ok || !slices.Contains(e.items("P31"), videoGame) {
			continue
		}
		score := 0
		if slices.ContainsFunc(e.items("P400"), func(p string) bool { return slices.Contains(want, p) }) {
			score += 4
		}
		if norm(search.Search[i].Label) == norm(name) || norm(e.Labels["en"].Value) == norm(name) {
			score += 2
		}
		// sem plataforma nem nome igual não confia (pode ser outro jogo)
		if score == 0 {
			continue
		}
		if score > bestScore {
			best, bestScore = e, score
		}
	}
	if bestScore < 0 {
		return Result{}, ErrNotFound
	}
	res := Result{Entity: best.ID, Title: best.label(), Year: best.year(), Genres: []string{}, Source: "https://www.wikidata.org/wiki/" + best.ID}
	if genres := best.items("P136"); len(genres) > 0 {
		if len(genres) > 4 {
			genres = genres[:4]
		}
		names, err := c.entities(ctx, genres, "labels")
		if err != nil {
			return Result{}, err
		}
		for _, g := range genres {
			if l := Genre(names[g].label()); l != "" && !slices.Contains(res.Genres, l) {
				res.Genres = append(res.Genres, l)
			}
		}
	}
	for _, lang := range []string{"pt", "en"} {
		title := best.Sitelinks[lang+"wiki"].Title
		if title == "" {
			continue
		}
		var sum struct {
			Type        string `json:"type"`
			Extract     string `json:"extract"`
			ContentURLs struct {
				Desktop struct {
					Page string `json:"page"`
				} `json:"desktop"`
			} `json:"content_urls"`
		}
		if _, err := c.get(ctx, c.summaryURL(lang, title), &sum); err != nil {
			return Result{}, err
		}
		if sum.Type == "disambiguation" || strings.TrimSpace(sum.Extract) == "" {
			continue
		}
		res.Summary = clip(strings.TrimSpace(spaces.ReplaceAllString(sum.Extract, " ")), 4000)
		if sum.ContentURLs.Desktop.Page != "" {
			res.Source = sum.ContentURLs.Desktop.Page
		}
		break
	}
	return res, nil
}

// Genre limpa o rótulo do Wikidata: "jogo eletrônico de tiro" vira "Tiro".
func Genre(label string) string {
	l := strings.TrimSpace(label)
	for _, p := range []string{"jogo eletrônico de ", "jogo de computador de ", "jogo de ", "jogo eletrônico "} {
		if strings.HasPrefix(strings.ToLower(l), p) {
			l = l[len(p):]
			break
		}
	}
	l = strings.TrimSuffix(strings.TrimSuffix(l, " video game"), " game")
	if l == "" || utf8.RuneCountInString(l) > 40 {
		return ""
	}
	r, n := utf8.DecodeRuneInString(l)
	return string(unicode.ToUpper(r)) + l[n:]
}

func clip(s string, max int) string {
	if utf8.RuneCountInString(s) <= max {
		return s
	}
	r := []rune(s)
	return strings.TrimSpace(string(r[:max-1])) + "…"
}
