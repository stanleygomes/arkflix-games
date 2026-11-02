package core

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/stanleygomes/arkflix-games/app/internal/meta"
	"github.com/stanleygomes/arkflix-games/app/internal/wiki"
)

// missAgain: jogo que não achou na internet só é procurado de novo depois disso.
const missAgain = 30 * 24 * time.Hour

// wikiState: busca de descrição, ano e gêneros no Wikidata/Wikipedia.
type wikiState struct {
	agent string
	gate  *wiki.Gate // um intervalo só pra todas as buscas (editor e lote)
	fast  *wiki.Gate // servidor falso dos testes
	file  string     // jogos que não acharam (id -> quando)

	mu     sync.Mutex
	misses map[string]int64
}

func newWikiState(cache, version string) *wikiState {
	w := &wikiState{
		agent:  "arkflix-games/" + version + " (https://github.com/stanleygomes/arkflix-games)",
		gate:   &wiki.Gate{},
		fast:   &wiki.Gate{Gap: time.Millisecond},
		file:   filepath.Join(cache, "wiki-misses.json"),
		misses: map[string]int64{},
	}
	if raw, err := os.ReadFile(w.file); err == nil {
		json.Unmarshal(raw, &w.misses)
	}
	return w
}

// client: ARKFLIX_WIKI_BASE troca os servidores (falso nos testes de ponta a
// ponta). Lido a cada uso: no modo de teste o endereço só existe depois do New.
func (w *wikiState) client() *wiki.Client {
	base := os.Getenv("ARKFLIX_WIKI_BASE")
	gate := w.gate
	if base != "" {
		gate = w.fast
	}
	return &wiki.Client{Base: base, UserAgent: w.agent, Gate: gate}
}

func (w *wikiState) missed(id string, now time.Time) bool {
	w.mu.Lock()
	defer w.mu.Unlock()
	at, ok := w.misses[id]
	return ok && now.Sub(time.Unix(at, 0)) < missAgain
}

func (w *wikiState) miss(id string, now time.Time) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.misses[id] = now.Unix()
}

func (w *wikiState) save() {
	w.mu.Lock()
	raw, _ := json.Marshal(w.misses)
	w.mu.Unlock()
	os.MkdirAll(filepath.Dir(w.file), 0o755)
	os.WriteFile(w.file, raw, 0o600)
}

// fillEmpty põe o que achou só nos campos vazios e nunca por cima de edição
// (campo editado, mesmo vazio, é escolha da pessoa).
func fillEmpty(edit meta.Override, auto meta.Info, r wiki.Result) (meta.Override, bool) {
	changed := false
	if edit.Summary == nil && auto.Summary == "" && r.Summary != "" {
		v := r.Summary
		edit.Summary, changed = &v, true
	}
	if edit.Year == nil && auto.Year == 0 && r.Year >= 1950 && r.Year <= 2100 {
		v := r.Year
		edit.Year, changed = &v, true
	}
	if edit.Genres == nil && len(auto.Genres) == 0 && len(r.Genres) > 0 {
		v := append([]string(nil), r.Genres...)
		edit.Genres, changed = &v, true
	}
	return edit, changed
}

// lacking: falta descrição, ano ou gênero (e dá pra completar).
func lacking(e entry) bool {
	if e.remote {
		return false
	}
	_, ok := fillEmpty(e.edit, e.auto, wiki.Result{Summary: "x", Year: 2000, Genres: []string{"x"}})
	return ok
}

func searchName(e entry) string {
	if e.info.Name != "" {
		return e.info.Name
	}
	return e.game.Title
}

// FetchMeta procura o jogo no Wikidata/Wikipedia e devolve o que achou, sem
// gravar (o editor preenche os campos vazios e mostra a fonte).
func (s *Service) FetchMeta(id string) (wiki.Result, error) {
	if err := s.adult(); err != nil {
		return wiki.Result{}, err
	}
	e, _, err := s.find(id)
	if err != nil {
		return wiki.Result{}, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	return s.wiki.client().Lookup(ctx, searchName(e), e.game.System)
}

// MissingMeta conta os jogos sem descrição, ano ou gênero que ainda vale
// procurar na internet (os que não acharam esperam 30 dias).
func (s *Service) MissingMeta() int {
	u, now := s.cur(), time.Now()
	s.mu.Lock()
	defer s.mu.Unlock()
	n := 0
	for _, e := range s.entries {
		if u.allows(e) && lacking(e) && !s.wiki.missed(e.game.ID, now) {
			n++
		}
	}
	return n
}

// CompleteMeta completa em segundo plano (tarefa "meta", cancelável) os
// jogos sem descrição, ano ou gênero, gravando como edição só nos campos vazios.
func (s *Service) CompleteMeta() error {
	if err := s.adult(); err != nil {
		return err
	}
	now := time.Now()
	var todo []entry
	s.mu.Lock()
	for _, e := range s.entries {
		if lacking(e) && !s.wiki.missed(e.game.ID, now) {
			todo = append(todo, e)
		}
	}
	s.mu.Unlock()
	if len(todo) == 0 {
		return errors.New("nenhum jogo com informação faltando pra procurar")
	}
	client := s.wiki.client()
	ok := s.runTaskCtx("meta", "Completando informações", func(ctx context.Context, progress func(float64)) (string, error) {
		found, missing := 0, 0
		defer func() {
			s.wiki.save()
			if found > 0 {
				s.opt.Emitter.Emit("library", s.views())
			}
		}()
		for i, e := range todo {
			if err := ctx.Err(); err != nil {
				return "", err
			}
			r, err := client.Lookup(ctx, searchName(e), e.game.System)
			switch {
			case errors.Is(err, wiki.ErrNotFound):
				s.wiki.miss(e.game.ID, time.Now())
				missing++
			case err != nil:
				return "", err
			default:
				filled, err := s.fillFromWiki(e.game.ID, r)
				if err != nil {
					return "", err
				}
				if filled {
					found++
				} else {
					s.wiki.miss(e.game.ID, time.Now())
					missing++
				}
			}
			progress(float64(i+1) / float64(len(todo)))
		}
		msg := fmt.Sprintf("%d jogo(s) completado(s)", found)
		if missing > 0 {
			msg += fmt.Sprintf(", %d sem informação na Wikipedia", missing)
		}
		return msg, nil
	})
	if !ok {
		return errors.New("já estou completando as informações")
	}
	return nil
}

// fillFromWiki relê a edição (pode ter mudado no editor enquanto buscava) e
// grava só os campos vazios.
func (s *Service) fillFromWiki(id string, r wiki.Result) (bool, error) {
	s.mu.Lock()
	var auto meta.Info
	found := false
	for _, e := range s.entries {
		if e.game.ID == id {
			auto, found = e.auto, true
			break
		}
	}
	s.mu.Unlock()
	if !found {
		return false, nil // saiu da biblioteca no meio
	}
	edit, changed := fillEmpty(s.over.Get(id), auto, r)
	if !changed {
		return false, nil
	}
	saved, err := s.over.Set(id, edit)
	if err != nil {
		return false, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for i, e := range s.entries {
		if e.game.ID == id {
			s.entries[i].edit = saved
			s.entries[i].info = saved.Apply(e.auto)
		}
	}
	return true, nil
}
