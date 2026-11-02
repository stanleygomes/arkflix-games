// Package lists guarda as listas pessoais de jogos ("Futebol", "Pra jogar a
// dois") em lists.json. Jogo apagado do disco continua no arquivo, mas a
// interface só mostra os que existem: não dá erro.
package lists

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode/utf8"
)

// List é uma lista pessoal. Com Rule, é automática: o conteúdo é calculado
// pela interface (fonte + filtros + ordem) e Games fica vazio.
type List struct {
	ID      string   `json:"id"`
	Name    string   `json:"name"`
	Games   []string `json:"games"`
	Created int64    `json:"created"`
	Rule    *Rule    `json:"rule,omitempty"`
}

// Rule é a regra de uma lista automática: os mesmos filtros e a mesma ordem
// da tela da biblioteca, aplicados a uma fonte (console ou coleção).
type Rule struct {
	Source  string   `json:"source"`  // system:snes, collection:all...
	Filters []string `json:"filters"` // fav, never, genre:RPG, decade:1990...
	Sort    string   `json:"sort"`    // name, yearAsc, plays...
}

// ErrSmart: lista automática não aceita jogo posto na mão.
var ErrSmart = errors.New("lista automática: os jogos entram pela regra, não na mão")

var (
	sourceRe = regexp.MustCompile(`^(system|collection):[a-z0-9_-]{1,30}$`)
	sorts    = []string{"name", "yearAsc", "yearDesc", "plays", "recent", "playTime", "rating", "added"}
	quick    = []string{"fav", "never", "saves", "completed"}
)

// Validate limpa e confere a regra.
func (r *Rule) Validate() error {
	if !sourceRe.MatchString(r.Source) {
		return errors.New("fonte da lista automática inválida")
	}
	if r.Sort == "" {
		r.Sort = "name"
	}
	if !slices.Contains(sorts, r.Sort) {
		return errors.New("ordem da lista automática inválida")
	}
	var clean []string
	for _, f := range r.Filters {
		f = strings.TrimSpace(f)
		if f == "" || slices.Contains(clean, f) {
			continue
		}
		if !validFilter(f) {
			return errors.New("filtro inválido: " + f)
		}
		clean = append(clean, f)
	}
	if len(clean) == 0 {
		return errors.New("lista automática precisa de pelo menos um filtro")
	}
	if len(clean) > 30 {
		return errors.New("filtros demais (máximo 30)")
	}
	r.Filters = clean
	return nil
}

func validFilter(f string) bool {
	kind, value, ok := strings.Cut(f, ":")
	if !ok {
		return slices.Contains(quick, f)
	}
	switch kind {
	case "players":
		return value == "1" || value == "2"
	case "genre":
		return value != "" && utf8.RuneCountInString(value) <= 40
	case "decade":
		n, err := strconv.Atoi(value)
		return err == nil && n >= 1950 && n <= 2100 && n%10 == 0
	}
	return false
}

// Store lê e grava as listas.
type Store struct {
	Path string
	mu   sync.Mutex
}

// ErrNotFound: lista não existe.
var ErrNotFound = errors.New("lista não encontrada")

func (s *Store) load() ([]List, error) {
	data, err := os.ReadFile(s.Path)
	if errors.Is(err, os.ErrNotExist) {
		return []List{}, nil
	}
	if err != nil {
		return nil, err
	}
	var out []List
	if err := json.Unmarshal(data, &out); err != nil {
		return nil, err
	}
	for i := range out {
		if out[i].Games == nil {
			out[i].Games = []string{}
		}
	}
	return out, nil
}

func (s *Store) save(all []List) error {
	data, _ := json.MarshalIndent(all, "", "  ")
	tmp := s.Path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, s.Path)
}

// All lista as listas na ordem em que foram criadas.
func (s *Store) All() ([]List, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.load()
}

func cleanName(name string, all []List, skip string) (string, error) {
	name = strings.Join(strings.Fields(name), " ")
	if name == "" {
		return "", errors.New("dê um nome pra lista")
	}
	if utf8.RuneCountInString(name) > 40 {
		return "", errors.New("nome grande demais (máximo 40 letras)")
	}
	for _, l := range all {
		if l.ID != skip && strings.EqualFold(l.Name, name) {
			return "", errors.New("já existe uma lista com esse nome")
		}
	}
	return name, nil
}

func newID() string {
	b := make([]byte, 6)
	rand.Read(b)
	return hex.EncodeToString(b)
}

func (s *Store) update(fn func([]List) ([]List, List, error)) (List, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	all, err := s.load()
	if err != nil {
		return List{}, err
	}
	all, out, err := fn(all)
	if err != nil {
		return List{}, err
	}
	return out, s.save(all)
}

// Create cria uma lista vazia.
func (s *Store) Create(name string) (List, error) {
	return s.update(func(all []List) ([]List, List, error) {
		clean, err := cleanName(name, all, "")
		if err != nil {
			return nil, List{}, err
		}
		l := List{ID: newID(), Name: clean, Games: []string{}, Created: time.Now().Unix()}
		return append(all, l), l, nil
	})
}

// CreateSmart cria uma lista automática a partir da regra.
func (s *Store) CreateSmart(name string, rule Rule) (List, error) {
	if err := rule.Validate(); err != nil {
		return List{}, err
	}
	return s.update(func(all []List) ([]List, List, error) {
		clean, err := cleanName(name, all, "")
		if err != nil {
			return nil, List{}, err
		}
		l := List{ID: newID(), Name: clean, Games: []string{}, Created: time.Now().Unix(), Rule: &rule}
		return append(all, l), l, nil
	})
}

func find(all []List, id string) int {
	return slices.IndexFunc(all, func(l List) bool { return l.ID == id })
}

// Rename troca o nome.
func (s *Store) Rename(id, name string) (List, error) {
	return s.update(func(all []List) ([]List, List, error) {
		i := find(all, id)
		if i < 0 {
			return nil, List{}, ErrNotFound
		}
		clean, err := cleanName(name, all, id)
		if err != nil {
			return nil, List{}, err
		}
		all[i].Name = clean
		return all, all[i], nil
	})
}

// Delete apaga a lista (os jogos continuam na biblioteca).
func (s *Store) Delete(id string) error {
	_, err := s.update(func(all []List) ([]List, List, error) {
		i := find(all, id)
		if i < 0 {
			return nil, List{}, ErrNotFound
		}
		return slices.Delete(all, i, i+1), List{}, nil
	})
	return err
}

// Set põe ou tira o jogo da lista.
func (s *Store) Set(id, game string, in bool) (List, error) {
	return s.update(func(all []List) ([]List, List, error) {
		i := find(all, id)
		if i < 0 {
			return nil, List{}, ErrNotFound
		}
		if all[i].Rule != nil {
			return nil, List{}, ErrSmart
		}
		has := slices.Contains(all[i].Games, game)
		switch {
		case in && !has:
			all[i].Games = append(all[i].Games, game)
		case !in && has:
			all[i].Games = slices.DeleteFunc(all[i].Games, func(g string) bool { return g == game })
		}
		return all, all[i], nil
	})
}
