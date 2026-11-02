package core

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"unicode/utf8"
)

// Capturas e gravações do RetroArch (pasta do perfil, configurada no
// guideCfg) e a anotação de cada jogo.

func (u *user) mediaDir(kind string) string { return filepath.Join(u.dir, "media", kind) }

// MediaFile é uma captura ou gravação de um jogo.
type MediaFile struct {
	ID   string `json:"id"`   // token pra /media/shot/<id>
	Kind string `json:"kind"` // image ou video
	At   int64  `json:"at"`
	Name string `json:"name"`
}

var mediaExt = map[string]string{".png": "image", ".jpg": "image", ".mkv": "video", ".mp4": "video", ".webm": "video"}

// GameMedia lista as capturas e gravações do jogo, da mais nova pra mais
// velha. O RetroArch nomeia "<nome da ROM>-<data>.<ext>".
func (s *Service) GameMedia(id string) ([]MediaFile, error) {
	e, _, err := s.find(id)
	if err != nil {
		return nil, err
	}
	base := filepath.Base(e.game.Launch)
	stem := strings.TrimSuffix(base, filepath.Ext(base)) + "-"
	u := s.cur()
	out := []MediaFile{}
	for _, kind := range []string{"screenshots", "videos"} {
		dir := u.mediaDir(kind)
		entries, _ := os.ReadDir(dir)
		for _, d := range entries {
			k := mediaExt[strings.ToLower(filepath.Ext(d.Name()))]
			if d.IsDir() || k == "" || !strings.HasPrefix(d.Name(), stem) {
				continue
			}
			info, err := d.Info()
			if err != nil {
				continue
			}
			out = append(out, MediaFile{ID: encodeToken(filepath.Join(dir, d.Name())), Kind: k, At: info.ModTime().Unix(), Name: d.Name()})
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].At > out[j].At })
	return out, nil
}

// serveShot entrega a captura/gravação: só de dentro da pasta de mídia do
// perfil em uso.
func (s *Service) serveShot(w http.ResponseWriter, r *http.Request, token string) {
	raw, err := base64.RawURLEncoding.DecodeString(token)
	path := string(raw)
	root := filepath.Join(s.cur().dir, "media") + string(filepath.Separator)
	if err != nil || !strings.HasPrefix(filepath.Clean(path), root) || mediaExt[strings.ToLower(filepath.Ext(path))] == "" {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Cache-Control", "no-cache")
	http.ServeFile(w, r, path)
}

// ---- anotação por jogo (notes.json do perfil)

var notesMu sync.Mutex

const noteMax = 2000

func (u *user) notesFile() string { return filepath.Join(u.dir, "notes.json") }

func (u *user) notes() map[string]string {
	out := map[string]string{}
	if b, err := os.ReadFile(u.notesFile()); err == nil {
		json.Unmarshal(b, &out)
	}
	return out
}

func (u *user) note(game string) string {
	notesMu.Lock()
	defer notesMu.Unlock()
	return u.notes()[game]
}

// GameNote devolve a anotação do jogo (deste perfil).
func (s *Service) GameNote(id string) string { return s.cur().note(id) }

// SetGameNote grava a anotação ("" apaga). Cada perfil tem as suas.
func (s *Service) SetGameNote(id, text string) error {
	if _, _, err := s.find(id); err != nil {
		return err
	}
	text = strings.TrimSpace(text)
	if utf8.RuneCountInString(text) > noteMax {
		return errors.New("anotação grande demais (até 2000 letras)")
	}
	u := s.cur()
	notesMu.Lock()
	defer notesMu.Unlock()
	all := u.notes()
	if text == "" {
		delete(all, id)
	} else {
		all[id] = text
	}
	b, err := json.Marshal(all)
	if err != nil {
		return err
	}
	return os.WriteFile(u.notesFile(), b, 0o644)
}
