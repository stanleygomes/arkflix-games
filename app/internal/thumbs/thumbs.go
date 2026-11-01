// Package thumbs baixa capas, prints e telas de título do acervo aberto do
// RetroArch (thumbnails.libretro.com), sem conta nem chave. O jogo é
// identificado pelo banco do libretro (DATs No-Intro/Redump/FBNeo): pelo nome
// do zip (arcade), pelo serial do disco (PS1/PS2/PSP), pelo CRC32 da ROM
// (cartuchos) e, se nada disso bater, pelo título aproximado.
package thumbs

import (
	"archive/zip"
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"hash/crc32"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode"

	"golang.org/x/text/unicode/norm"

	"github.com/stanleygomes/arkflix-games/app/internal/isoinfo"
	"github.com/stanleygomes/arkflix-games/app/internal/library"
	"github.com/stanleygomes/arkflix-games/app/internal/meta"
)

// Padrões dos servidores (configuráveis pra teste).
const (
	DefaultImages = "https://thumbnails.libretro.com"
	DefaultDB     = "https://raw.githubusercontent.com/libretro/libretro-database/master/metadat"
)

// platform liga o console ao DAT e à pasta de imagens.
type platform struct {
	dat    string // caminho relativo do DAT
	images string // pasta no servidor de imagens
	zip    bool   // identifica pelo nome do zip (arcade)
}

var platforms = map[string]platform{
	"nes":     {"no-intro/Nintendo - Nintendo Entertainment System.dat", "Nintendo - Nintendo Entertainment System", false},
	"snes":    {"no-intro/Nintendo - Super Nintendo Entertainment System.dat", "Nintendo - Super Nintendo Entertainment System", false},
	"genesis": {"no-intro/Sega - Mega Drive - Genesis.dat", "Sega - Mega Drive - Genesis", false},
	"gbc":     {"no-intro/Nintendo - Game Boy Color.dat", "Nintendo - Game Boy Color", false},
	"n64":     {"no-intro/Nintendo - Nintendo 64.dat", "Nintendo - Nintendo 64", false},
	"nds":     {"no-intro/Nintendo - Nintendo DS.dat", "Nintendo - Nintendo DS", false},
	"psx":     {"redump/Sony - PlayStation.dat", "Sony - PlayStation", false},
	"ps2":     {"redump/Sony - PlayStation 2.dat", "Sony - PlayStation 2", false},
	"psp":     {"redump/Sony - PlayStation Portable.dat", "Sony - PlayStation Portable", false},
	"arcade":  {"fbneo-split/FBNeo - Arcade Games.dat", "FBNeo - Arcade Games", true},
}

// Supported diz se há capas pro console.
func Supported(system string) bool { _, ok := platforms[system]; return ok }

// Entry é um jogo do DAT.
type Entry struct {
	Name   string
	Serial string
	Roms   []Rom
}

// Rom é um arquivo do jogo no DAT.
type Rom struct {
	Name string
	Size int64
	CRC  uint32
}

var (
	reGameName = regexp.MustCompile(`^\s*name\s+"(.*)"\s*$`)
	reSerial   = regexp.MustCompile(`^\s*serial\s+"(.*)"\s*$`)
	reRom      = regexp.MustCompile(`^\s*rom\s*\(\s*name\s+(?:"([^"]*)"|(\S+))(.*)\)\s*$`)
	reSize     = regexp.MustCompile(`\bsize\s+(\d+)`)
	reCRC      = regexp.MustCompile(`\bcrc\s+([0-9A-Fa-f]{8})`)
)

// ParseDAT lê um DAT no formato clrmamepro.
func ParseDAT(r io.Reader) ([]Entry, error) {
	var out []Entry
	var cur *Entry
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 64*1024), 4<<20)
	for sc.Scan() {
		line := sc.Text()
		trim := strings.TrimSpace(line)
		switch {
		case strings.HasPrefix(trim, "game (") || strings.HasPrefix(trim, "machine ("):
			out = append(out, Entry{})
			cur = &out[len(out)-1]
		case cur == nil:
		case trim == ")":
			cur = nil
		case reRom.MatchString(trim):
			m := reRom.FindStringSubmatch(trim)
			rom := Rom{Name: m[1]}
			if rom.Name == "" {
				rom.Name = m[2]
			}
			if sm := reSize.FindStringSubmatch(m[3]); sm != nil {
				rom.Size, _ = strconv.ParseInt(sm[1], 10, 64)
			}
			if cm := reCRC.FindStringSubmatch(m[3]); cm != nil {
				v, _ := strconv.ParseUint(cm[1], 16, 32)
				rom.CRC = uint32(v)
			}
			cur.Roms = append(cur.Roms, rom)
		case reGameName.MatchString(trim) && cur.Name == "":
			cur.Name = reGameName.FindStringSubmatch(trim)[1]
		case reSerial.MatchString(trim):
			cur.Serial = reSerial.FindStringSubmatch(trim)[1]
		}
	}
	return out, sc.Err()
}

// Index acha jogos do DAT por CRC, serial, zip ou título.
type Index struct {
	byCRC    map[uint32]string
	bySerial map[string]string
	byZip    map[string]string
	byTitle  map[string][]string
}

func serialKey(s string) string {
	return strings.ToUpper(strings.NewReplacer("-", "", "_", "", ".", "", " ", "").Replace(s))
}

// NewIndex monta os índices.
func NewIndex(entries []Entry) *Index {
	ix := &Index{byCRC: map[uint32]string{}, bySerial: map[string]string{}, byZip: map[string]string{}, byTitle: map[string][]string{}}
	for _, e := range entries {
		if e.Name == "" {
			continue
		}
		if e.Serial != "" {
			for _, s := range strings.Split(e.Serial, ",") {
				if k := serialKey(s); k != "" {
					if _, dup := ix.bySerial[k]; !dup {
						ix.bySerial[k] = e.Name
					}
				}
			}
		}
		for _, r := range e.Roms {
			if r.CRC != 0 {
				if _, dup := ix.byCRC[r.CRC]; !dup {
					ix.byCRC[r.CRC] = e.Name
				}
			}
			if strings.HasSuffix(strings.ToLower(r.Name), ".zip") {
				ix.byZip[strings.ToLower(r.Name)] = e.Name
			}
		}
		k := TitleKey(e.Name)
		ix.byTitle[k] = append(ix.byTitle[k], e.Name)
	}
	return ix
}

var (
	reTags    = regexp.MustCompile(`\s*[\(\[][^\)\]]*[\)\]]`)
	reNonWord = regexp.MustCompile(`[^a-z0-9]+`)
)

// TitleKey normaliza o título pra comparação: sem tags, acentos, pontuação e
// artigos ("The Legend of Zelda" = "Legend of Zelda, The").
func TitleKey(name string) string {
	s := reTags.ReplaceAllString(name, "")
	s = norm.NFD.String(s)
	s = strings.Map(func(r rune) rune {
		if unicode.Is(unicode.Mn, r) {
			return -1
		}
		return unicode.ToLower(r)
	}, s)
	s = strings.ReplaceAll(s, "&", " and ")
	s = reNonWord.ReplaceAllString(s, " ")
	words := strings.Fields(s)
	var keep []string
	for _, w := range words {
		if w != "the" && w != "a" {
			keep = append(keep, w)
		}
	}
	return strings.Join(keep, " ")
}

// rank prefere EUA, depois mundo, Europa, Brasil, Japão; sem beta/demo/protótipo.
func rank(name string) int {
	lower := strings.ToLower(name)
	score := 50
	for i, region := range []string{"(usa", "(world", "(europe", "(brazil", "(japan"} {
		if strings.Contains(lower, region) {
			score = i * 10
			break
		}
	}
	for _, bad := range []string{"beta", "proto", "demo", "sample", "virtual console", "pirate", "hack", "unl)", "kiosk"} {
		if strings.Contains(lower, bad) {
			score += 100
		}
	}
	return score*1000 + len(name)
}

// ByTitle escolhe a melhor entrada do DAT pro título.
func (ix *Index) ByTitle(title string) string {
	cands := ix.byTitle[TitleKey(title)]
	if len(cands) == 0 {
		return ""
	}
	sorted := append([]string(nil), cands...)
	sort.Slice(sorted, func(i, j int) bool { return rank(sorted[i]) < rank(sorted[j]) })
	return sorted[0]
}

// Match é o resultado da identificação.
type Match struct {
	Name string `json:"name"` // nome canônico do DAT (= nome do arquivo de imagem)
	How  string `json:"how"`  // zip, serial, crc, title
}

// romCRC calcula o CRC32 do conteúdo (zip: o do maior arquivo, sem descompactar).
// SNES com cabeçalho de copiadora (512 bytes) e NES com cabeçalho iNES dão
// também o CRC sem cabeçalho.
func romCRC(path string) []uint32 {
	if strings.EqualFold(filepath.Ext(path), ".zip") {
		z, err := zip.OpenReader(path)
		if err != nil {
			return nil
		}
		defer z.Close()
		var best *zip.File
		for _, f := range z.File {
			if !f.FileInfo().IsDir() && (best == nil || f.UncompressedSize64 > best.UncompressedSize64) {
				best = f
			}
		}
		if best == nil {
			return nil
		}
		out := []uint32{best.CRC32}
		if best.UncompressedSize64%1024 == 512 || best.UncompressedSize64%1024 == 16 {
			if rc, err := best.Open(); err == nil {
				data, _ := io.ReadAll(io.LimitReader(rc, 128<<20))
				rc.Close()
				out = append(out, headerless(data)...)
			}
		}
		return out
	}
	info, err := os.Stat(path)
	if err != nil || info.Size() > 128<<20 {
		return nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	return append([]uint32{crc32.ChecksumIEEE(data)}, headerless(data)...)
}

func headerless(data []byte) []uint32 {
	switch {
	case len(data)%1024 == 512:
		return []uint32{crc32.ChecksumIEEE(data[512:])}
	case len(data) > 16 && string(data[:4]) == "NES\x1a":
		return []uint32{crc32.ChecksumIEEE(data[16:])}
	}
	return nil
}

// Identify acha o nome canônico do jogo.
func Identify(ix *Index, system string, g library.Game) Match {
	p := platforms[system]
	if p.zip {
		if name := ix.byZip[strings.ToLower(filepath.Base(g.Launch))]; name != "" {
			return Match{name, "zip"}
		}
	}
	switch system {
	case "psx", "ps2", "psp":
		disc := g.Launch
		if strings.EqualFold(filepath.Ext(disc), ".cue") {
			if bins, _ := filepath.Glob(filepath.Join(filepath.Dir(disc), "*.bin")); len(bins) > 0 {
				sort.Strings(bins)
				disc = bins[0]
			}
		}
		if info, err := isoinfo.ReadFile(disc); err == nil && info.Serial != "" {
			if name := ix.bySerial[serialKey(info.Serial)]; name != "" {
				return Match{name, "serial"}
			}
		}
	default:
		for _, crc := range romCRC(g.Launch) {
			if name := ix.byCRC[crc]; name != "" {
				return Match{name, "crc"}
			}
		}
	}
	for _, t := range []string{g.FileName, g.Title} {
		if name := ix.ByTitle(t); name != "" {
			return Match{name, "title"}
		}
	}
	return Match{}
}

// safeName troca os caracteres que o libretro não usa em nome de arquivo.
func safeName(name string) string {
	return strings.Map(func(r rune) rune {
		if strings.ContainsRune(`&*/:`+"`"+`<>?\|"`, r) {
			return '_'
		}
		return r
	}, name)
}

// Kinds são os tipos de imagem: capa, print e tela de título.
var Kinds = map[string]string{"cover": "Named_Boxarts", "snap": "Named_Snaps", "title": "Named_Titles"}

// Store baixa e guarda os DATs e as imagens.
type Store struct {
	Cache  string // ~/.cache/arkflix-games
	Images string // servidor de imagens
	DB     string // servidor dos DATs
	HTTP   *http.Client

	mu      sync.Mutex
	indexes map[string]*Index
}

func (s *Store) client() *http.Client {
	if s.HTTP != nil {
		return s.HTTP
	}
	return &http.Client{Timeout: 60 * time.Second}
}

// Configure troca os servidores (vem das configurações).
func (s *Store) Configure(images, db string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Images, s.DB = images, db
}

func (s *Store) servers() (string, string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return strings.TrimRight(s.Images, "/"), strings.TrimRight(s.DB, "/")
}

func (s *Store) get(ctx context.Context, rawURL, dest string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return err
	}
	resp, err := s.client().Do(req)
	if err != nil {
		return fmt.Errorf("sem conexão com %s: %w", req.URL.Host, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return os.ErrNotExist
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("%s respondeu %d", req.URL.Host, resp.StatusCode)
	}
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(dest), ".dl-*")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())
	if _, err := io.Copy(tmp, io.LimitReader(resp.Body, 64<<20)); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), dest)
}

func escapePath(p string) string {
	parts := strings.Split(p, "/")
	for i, part := range parts {
		parts[i] = url.PathEscape(part)
	}
	return strings.Join(parts, "/")
}

// index devolve o índice do console, baixando o DAT se faltar ou tiver mais de 30 dias.
func (s *Store) index(ctx context.Context, system string) (*Index, error) {
	s.mu.Lock()
	if ix, ok := s.indexes[system]; ok {
		s.mu.Unlock()
		return ix, nil
	}
	s.mu.Unlock()
	p, ok := platforms[system]
	if !ok {
		return nil, errors.New("console sem capas no libretro")
	}
	path := filepath.Join(s.Cache, "dat", filepath.Base(p.dat))
	info, err := os.Stat(path)
	if err != nil || time.Since(info.ModTime()) > 30*24*time.Hour {
		_, db := s.servers()
		if err := s.get(ctx, db+"/"+escapePath(p.dat), path); err != nil && info == nil {
			return nil, fmt.Errorf("banco do libretro (%s): %w", system, err)
		}
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	entries, err := ParseDAT(f)
	if err != nil {
		return nil, err
	}
	ix := NewIndex(entries)
	s.mu.Lock()
	if s.indexes == nil {
		s.indexes = map[string]*Index{}
	}
	s.indexes[system] = ix
	s.mu.Unlock()
	return ix, nil
}

func (s *Store) gameDir(g library.Game) string { return filepath.Join(s.Cache, "media", g.ID) }

// Fetch identifica o jogo e baixa as imagens que existirem. Volta o que achou.
func (s *Store) Fetch(ctx context.Context, g library.Game) (Match, int, error) {
	p, ok := platforms[g.System]
	if !ok {
		return Match{}, 0, nil
	}
	ix, err := s.index(ctx, g.System)
	if err != nil {
		return Match{}, 0, err
	}
	m := Identify(ix, g.System, g)
	dir := s.gameDir(g)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return m, 0, err
	}
	data, _ := json.Marshal(m)
	os.WriteFile(filepath.Join(dir, "match.json"), data, 0o644)
	if m.Name == "" {
		return m, 0, nil
	}
	got := 0
	images, _ := s.servers()
	for kind, folder := range Kinds {
		u := fmt.Sprintf("%s/%s/%s/%s.png", images, url.PathEscape(p.images), folder, url.PathEscape(safeName(m.Name)))
		err := s.get(ctx, u, filepath.Join(dir, kind+".png"))
		switch {
		case err == nil:
			got++
		case errors.Is(err, os.ErrNotExist):
		default:
			return m, got, err
		}
	}
	return m, got, nil
}

// Fetched diz se o jogo já passou pelo download (achou ou não).
func (s *Store) Fetched(g library.Game) bool {
	_, err := os.Stat(filepath.Join(s.gameDir(g), "match.json"))
	return err == nil
}

// Image devolve o caminho da imagem em cache (ou vazio).
func (s *Store) Image(g library.Game, kind string) string {
	path := filepath.Join(s.gameDir(g), kind+".png")
	if info, err := os.Stat(path); err == nil && info.Size() > 0 {
		return path
	}
	return ""
}

// Provider junta os metadados locais com as imagens em cache. Não acessa a
// rede: quem baixa é o Store.Fetch (tarefa "Atualizar capas").
type Provider struct {
	Base interface {
		Lookup(g library.Game) meta.Info
		Refresh()
	}
	Store *Store
}

// Lookup usa a capa local se houver; senão, a do libretro.
func (p *Provider) Lookup(g library.Game) meta.Info {
	info := p.Base.Lookup(g)
	if info.Cover == "" {
		info.Cover = p.Store.Image(g, "cover")
	}
	if info.Snap == "" {
		info.Snap = p.Store.Image(g, "snap")
	}
	if info.TitleShot == "" {
		info.TitleShot = p.Store.Image(g, "title")
	}
	return info
}

// Refresh repassa pro provedor base.
func (p *Provider) Refresh() { p.Base.Refresh() }
