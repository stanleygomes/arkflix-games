// Package profiles guarda quem joga: nome, avatar, cor e PIN opcional.
//
// O perfil principal ("main") usa a pasta de dados raiz, então quem já tinha
// favoritos, listas e histórico continua com eles (migração sem mover nada).
// Os outros perfis ficam em profiles/<id>/. Apagar um perfil move a pasta pra
// profiles/apagados/ em vez de remover: saves não somem por engano.
package profiles

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"time"
	"unicode/utf8"
)

// Main é o perfil que existia antes dos perfis (dados na pasta raiz).
const Main = "main"

// Max é quantos perfis cabem na tela de escolha.
const Max = 8

// Characters são os personagens clássicos (imagens em app/web/public/avatars),
// em ordem de estreia; PixelAvatars, os desenhos em pixel art da interface.
var (
	Characters   = []string{"pac-man", "mario", "luigi", "donkey-kong", "mr-game-watch", "ice-climbers", "duck-hunt", "rob", "link", "zelda", "samus", "simon", "little-mac", "mega-man", "peach", "bowser", "ness", "kirby", "sonic", "yoshi", "fox", "wario", "ryu", "pikachu", "jigglypuff", "mewtwo", "captain-falcon", "diddy-kong", "king-k-rool", "ganondorf", "cloud", "snake", "terry", "banjo-kazooie"}
	PixelAvatars = []string{"knight", "mage", "ghost", "cat", "robot", "mushroom", "alien", "dragon"}
	Avatars      = append(append([]string{}, Characters...), PixelAvatars...)
)

// Colors são as cores de fundo do avatar.
var Colors = []string{"#d94848", "#e8902a", "#e8c838", "#44a95a", "#3a86d8", "#8a5ad0", "#d8579a", "#2fb3b3"}

var (
	ErrNotFound = errors.New("perfil não encontrado")
	ErrPIN      = errors.New("PIN errado")
)

// Profile é um perfil gravado. PIN guarda sal e hash, nunca o número.
type Profile struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Avatar string `json:"avatar"`
	Color  string `json:"color"`
	Kids   bool   `json:"kids"`
	// DailyMinutes e Bedtime só valem no perfil infantil: minutos de jogo por
	// dia (0 = sem limite) e hora de dormir "HH:MM" (vazio = sem).
	DailyMinutes int    `json:"dailyMinutes,omitempty"`
	Bedtime      string `json:"bedtime,omitempty"`
	PIN          string `json:"pin,omitempty"`
	Created      int64  `json:"created"`
	// New: o principal criado sozinho na primeira vez, ainda sem nome e
	// avatar escolhidos (a interface abre a configuração do perfil).
	New bool `json:"new,omitempty"`
}

// View é o perfil como a interface vê (sem o hash do PIN).
type View struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Avatar string `json:"avatar"`
	Color  string `json:"color"`
	Kids   bool   `json:"kids"`
	HasPIN bool   `json:"hasPIN"`
	Main   bool   `json:"main"`

	DailyMinutes int    `json:"dailyMinutes"`
	Bedtime      string `json:"bedtime"`
}

func (p Profile) View() View {
	return View{ID: p.ID, Name: p.Name, Avatar: p.Avatar, Color: p.Color, Kids: p.Kids, HasPIN: p.PIN != "", Main: p.ID == Main,
		DailyMinutes: p.DailyMinutes, Bedtime: p.Bedtime}
}

// Store lê e grava profiles.json na pasta de dados.
type Store struct {
	Dir  string
	mu   sync.Mutex
	data []Profile
}

func (s *Store) path() string { return filepath.Join(s.Dir, "profiles.json") }

func (s *Store) load() error {
	if s.data != nil {
		return nil
	}
	raw, err := os.ReadFile(s.path())
	if errors.Is(err, os.ErrNotExist) {
		s.data = []Profile{{ID: Main, Name: "Jogador 1", Avatar: "mario", Color: Colors[4], Created: time.Now().Unix(), New: true}}
		return nil
	}
	if err != nil {
		return err
	}
	var data []Profile
	if err := json.Unmarshal(raw, &data); err != nil {
		return fmt.Errorf("profiles.json: %w", err)
	}
	if !slices.ContainsFunc(data, func(p Profile) bool { return p.ID == Main }) {
		return errors.New("profiles.json sem o perfil principal")
	}
	s.data = data
	return nil
}

func (s *Store) save() error {
	raw, _ := json.MarshalIndent(s.data, "", "  ")
	tmp := s.path() + ".tmp"
	if err := os.WriteFile(tmp, raw, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, s.path())
}

// All devolve os perfis na ordem de criação.
func (s *Store) All() ([]Profile, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.load(); err != nil {
		return nil, err
	}
	return slices.Clone(s.data), nil
}

// Get devolve um perfil.
func (s *Store) Get(id string) (Profile, error) {
	all, err := s.All()
	if err != nil {
		return Profile{}, err
	}
	for _, p := range all {
		if p.ID == id {
			return p, nil
		}
	}
	return Profile{}, ErrNotFound
}

// DataDir é onde ficam estado, listas, histórico, controle e saves do perfil.
func (s *Store) DataDir(id string) string {
	if id == Main {
		return s.Dir
	}
	return filepath.Join(s.Dir, "profiles", id)
}

func (s *Store) validate(id, name, avatar, color string) (string, error) {
	name = strings.Join(strings.Fields(name), " ")
	if name == "" {
		return "", errors.New("o perfil precisa de um nome")
	}
	if utf8.RuneCountInString(name) > 20 {
		return "", errors.New("nome grande demais (máximo 20 letras)")
	}
	if !slices.Contains(Avatars, avatar) {
		return "", errors.New("avatar desconhecido")
	}
	if !slices.Contains(Colors, color) {
		return "", errors.New("cor desconhecida")
	}
	for _, p := range s.data {
		if p.ID != id && strings.EqualFold(p.Name, name) {
			return "", fmt.Errorf("já existe um perfil chamado %q", p.Name)
		}
	}
	return name, nil
}

func (s *Store) find(id string) int {
	return slices.IndexFunc(s.data, func(p Profile) bool { return p.ID == id })
}

// Create cria um perfil novo (sem PIN).
func (s *Store) Create(name, avatar, color string) (Profile, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.load(); err != nil {
		return Profile{}, err
	}
	if len(s.data) >= Max {
		return Profile{}, fmt.Errorf("no máximo %d perfis", Max)
	}
	name, err := s.validate("", name, avatar, color)
	if err != nil {
		return Profile{}, err
	}
	buf := make([]byte, 4)
	rand.Read(buf)
	p := Profile{ID: hex.EncodeToString(buf), Name: name, Avatar: avatar, Color: color, Created: time.Now().Unix()}
	if err := os.MkdirAll(s.DataDir(p.ID), 0o700); err != nil {
		return Profile{}, err
	}
	s.data = append(s.data, p)
	return p, s.save()
}

// Update troca nome, avatar e cor.
func (s *Store) Update(id, name, avatar, color string) (Profile, error) {
	return s.change(id, func(p *Profile) error {
		clean, err := s.validate(id, name, avatar, color)
		if err != nil {
			return err
		}
		p.Name, p.Avatar, p.Color, p.New = clean, avatar, color, false
		return nil
	})
}

// SetKids marca o perfil como infantil (o principal nunca é).
func (s *Store) SetKids(id string, kids bool) (Profile, error) {
	return s.change(id, func(p *Profile) error {
		if id == Main && kids {
			return errors.New("o perfil principal não pode ser infantil")
		}
		p.Kids = kids
		return nil
	})
}

// MaxDailyMinutes: o maior limite por dia (12 h).
const MaxDailyMinutes = 12 * 60

// SetLimits define o tempo por dia (0 = sem limite) e a hora de dormir
// ("HH:MM", vazio = sem) do perfil.
func (s *Store) SetLimits(id string, minutes int, bedtime string) (Profile, error) {
	if minutes < 0 || minutes > MaxDailyMinutes {
		return Profile{}, fmt.Errorf("o limite por dia vai de 0 a %d minutos", MaxDailyMinutes)
	}
	if bedtime != "" {
		if _, err := time.Parse("15:04", bedtime); err != nil || len(bedtime) != 5 {
			return Profile{}, errors.New("hora de dormir no formato HH:MM")
		}
	}
	return s.change(id, func(p *Profile) error {
		p.DailyMinutes, p.Bedtime = minutes, bedtime
		return nil
	})
}

// SetPIN troca o PIN; precisa do atual se já houver. next vazio remove.
func (s *Store) SetPIN(id, current, next string) (Profile, error) {
	return s.change(id, func(p *Profile) error {
		if !matches(p.PIN, current) {
			return ErrPIN
		}
		if next == "" {
			p.PIN = ""
			return nil
		}
		if !validPIN(next) {
			return errors.New("o PIN tem 4 números")
		}
		p.PIN = hashPIN(next)
		return nil
	})
}

func (s *Store) change(id string, fn func(*Profile) error) (Profile, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.load(); err != nil {
		return Profile{}, err
	}
	i := s.find(id)
	if i < 0 {
		return Profile{}, ErrNotFound
	}
	p := s.data[i]
	if err := fn(&p); err != nil {
		return Profile{}, err
	}
	s.data[i] = p
	return p, s.save()
}

// Check confere o PIN (perfil sem PIN aceita qualquer coisa).
func (s *Store) Check(id, pin string) error {
	p, err := s.Get(id)
	if err != nil {
		return err
	}
	if !matches(p.PIN, pin) {
		return ErrPIN
	}
	return nil
}

// Delete tira o perfil da lista e guarda a pasta dele em profiles/apagados/.
func (s *Store) Delete(id, pin string) error {
	if id == Main {
		return errors.New("o perfil principal não pode ser apagado")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.load(); err != nil {
		return err
	}
	i := s.find(id)
	if i < 0 {
		return ErrNotFound
	}
	if !matches(s.data[i].PIN, pin) {
		return ErrPIN
	}
	if dir := s.DataDir(id); exists(dir) {
		trash := filepath.Join(s.Dir, "profiles", "apagados")
		if err := os.MkdirAll(trash, 0o700); err != nil {
			return err
		}
		if err := os.Rename(dir, filepath.Join(trash, fmt.Sprintf("%s-%d", id, time.Now().Unix()))); err != nil {
			return err
		}
	}
	s.data = slices.Delete(s.data, i, i+1)
	return s.save()
}

func exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func validPIN(pin string) bool {
	if len(pin) != 4 {
		return false
	}
	for _, c := range pin {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}

func hashPIN(pin string) string {
	salt := make([]byte, 8)
	rand.Read(salt)
	return hex.EncodeToString(salt) + ":" + digest(salt, pin)
}

func digest(salt []byte, pin string) string {
	sum := sha256.Sum256(append(append([]byte{}, salt...), pin...))
	return hex.EncodeToString(sum[:])
}

func matches(stored, pin string) bool {
	if stored == "" {
		return true
	}
	saltHex, want, ok := strings.Cut(stored, ":")
	salt, err := hex.DecodeString(saltHex)
	if !ok || err != nil {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(digest(salt, pin)), []byte(want)) == 1
}
