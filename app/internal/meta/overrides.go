package meta

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"unicode/utf8"
)

// Override é o que a pessoa editou num jogo. Campo nil = automático (vem do
// ES-DE, do libretro ou do nome do arquivo). Nota, completo e quebrado só
// existem aqui.
type Override struct {
	Name      *string   `json:"name,omitempty"`
	Summary   *string   `json:"summary,omitempty"`
	Developer *string   `json:"developer,omitempty"`
	Players   *string   `json:"players,omitempty"`
	Year      *int      `json:"year,omitempty"`
	Genres    *[]string `json:"genres,omitempty"`
	Cover     string    `json:"cover,omitempty"` // imagem local escolhida
	Rating    int       `json:"rating"`          // 0 a 5 estrelas
	Completed bool      `json:"completed"`
	Broken    bool      `json:"broken"`
}

var imageExt = map[string]bool{".png": true, ".jpg": true, ".jpeg": true, ".webp": true, ".gif": true}

func limit(s *string, max int, label string) error {
	if s == nil {
		return nil
	}
	*s = strings.TrimSpace(*s)
	if utf8.RuneCountInString(*s) > max {
		return fmt.Errorf("%s grande demais (máximo %d letras)", label, max)
	}
	return nil
}

// Validate limpa e confere os campos.
func (o *Override) Validate() error {
	for _, f := range []struct {
		v     *string
		max   int
		label string
	}{{o.Name, 120, "nome"}, {o.Summary, 4000, "descrição"}, {o.Developer, 80, "desenvolvedora"}, {o.Players, 20, "jogadores"}} {
		if err := limit(f.v, f.max, f.label); err != nil {
			return err
		}
	}
	if o.Name != nil && *o.Name == "" {
		return errors.New("o nome não pode ficar vazio (use restaurar pra voltar ao automático)")
	}
	if o.Year != nil && *o.Year != 0 && (*o.Year < 1950 || *o.Year > 2100) {
		return errors.New("ano fora do intervalo (1950 a 2100)")
	}
	if o.Genres != nil {
		var clean []string
		for _, g := range *o.Genres {
			if g = strings.TrimSpace(g); g != "" {
				if utf8.RuneCountInString(g) > 40 {
					return errors.New("gênero grande demais (máximo 40 letras)")
				}
				clean = append(clean, g)
			}
		}
		if len(clean) > 10 {
			return errors.New("no máximo 10 gêneros")
		}
		if clean == nil {
			clean = []string{}
		}
		o.Genres = &clean
	}
	if o.Rating < 0 || o.Rating > 5 {
		return errors.New("nota vai de 0 a 5")
	}
	if o.Cover != "" {
		if !imageExt[strings.ToLower(filepath.Ext(o.Cover))] {
			return errors.New("a capa precisa ser uma imagem (png, jpg, webp ou gif)")
		}
		if info, err := os.Stat(o.Cover); err != nil || info.IsDir() {
			return errors.New("imagem da capa não encontrada")
		}
	}
	return nil
}

// Empty diz se não sobrou nada editado.
func (o Override) Empty() bool {
	return o == (Override{})
}

// Apply põe as edições por cima do automático.
func (o Override) Apply(info Info) Info {
	if o.Name != nil {
		info.Name = *o.Name
	}
	if o.Summary != nil {
		info.Summary = *o.Summary
	}
	if o.Developer != nil {
		info.Developer = *o.Developer
	}
	if o.Players != nil {
		info.Players = *o.Players
	}
	if o.Year != nil {
		info.Year = *o.Year
	}
	if o.Genres != nil {
		info.Genres = *o.Genres
	}
	if o.Cover != "" {
		info.Cover = o.Cover
	}
	return info
}

// Overrides guarda as edições em metadata.json (id do jogo -> edição).
type Overrides struct {
	Path string
	mu   sync.Mutex
	data map[string]Override
}

func (s *Overrides) load() error {
	if s.data != nil {
		return nil
	}
	s.data = map[string]Override{}
	raw, err := os.ReadFile(s.Path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	return json.Unmarshal(raw, &s.data)
}

// Get devolve a edição do jogo (zero = nada editado).
func (s *Overrides) Get(id string) Override {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.load() != nil {
		return Override{}
	}
	return s.data[id]
}

// Set valida e grava; edição vazia apaga a entrada.
func (s *Overrides) Set(id string, o Override) (Override, error) {
	if err := o.Validate(); err != nil {
		return Override{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.load(); err != nil {
		return Override{}, err
	}
	if o.Empty() {
		delete(s.data, id)
	} else {
		s.data[id] = o
	}
	raw, _ := json.MarshalIndent(s.data, "", "  ")
	tmp := s.Path + ".tmp"
	if err := os.WriteFile(tmp, raw, 0o600); err != nil {
		return Override{}, err
	}
	return o, os.Rename(tmp, s.Path)
}
