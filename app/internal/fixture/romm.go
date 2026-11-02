package fixture

import (
	"bytes"
	"encoding/json"
	"fmt"
	"image/color"
	"io"
	"maps"
	"net/http"
	"os"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"
)

// FakeRomM imita a API de um servidor RomM (o que o app usa): login, token de
// dispositivo, catálogo, download e saves/states. Usuário "stanley", senha
// "senha123".
type FakeRomM struct {
	mu       sync.Mutex
	firmware []fakeFirmware
	assets   map[string][]fakeAsset // "saves"/"states"
	nextID   int
	uploads  []string
	tokens   []string
	files    map[int][]byte // conteúdo dos jogos (id do jogo ou do arquivo)
	ranges   []string       // cabeçalhos Range recebidos nos downloads
	// Slow: espera entre pedaços de 16 bytes dos downloads (testes conseguem
	// pausar no meio). NewFakeRomM lê ARKFLIX_FAKE_ROMM_SLOW (milissegundos).
	Slow time.Duration
}

// SetRom troca o conteúdo de um jogo (ou arquivo de jogo) do servidor.
func (f *FakeRomM) SetRom(id int, data []byte) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.files[id] = data
}

// Ranges: cabeçalhos Range recebidos nos downloads de jogos.
func (f *FakeRomM) Ranges() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]string{}, f.ranges...)
}

// Uploads: envios recebidos, "<kind>:<rom>:<arquivo>".
func (f *FakeRomM) Uploads() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]string{}, f.uploads...)
}

// Tokens: nomes dos tokens de dispositivo criados.
func (f *FakeRomM) Tokens() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]string{}, f.tokens...)
}

type fakeAsset struct {
	ID       int       `json:"id"`
	RomID    int       `json:"rom_id"`
	FileName string    `json:"file_name"`
	Emulator string    `json:"emulator"`
	Size     int64     `json:"file_size_bytes"`
	Updated  time.Time `json:"updated_at"`
	data     []byte
}

const (
	fakeAccess = "acesso-curto"
	fakeDevice = "token-do-app"
)

// Arquivos dos jogos do servidor (cada FakeRomM começa com uma cópia).
var fakeRomFiles = map[int][]byte{
	1:  ZipBytes(map[string]string{"dkc.sfc": "SNES"}),
	21: []byte("FILE \"Spyro the Dragon (USA).bin\" BINARY\n"),
	22: []byte("PSX-DISCO"),
	3:  ZipBytes(map[string]string{"smw.sfc": "SNES"}),
}

func (f *FakeRomM) roms() []map[string]any {
	f.mu.Lock()
	defer f.mu.Unlock()
	fakeRomFiles := f.files
	rom := func(id int, name, fs, platform string, size int64, multi bool, files []map[string]any) map[string]any {
		if files == nil {
			files = []map[string]any{}
		}
		return map[string]any{
			"id": id, "name": name, "fs_name": fs, "platform_fs_slug": platform, "fs_size_bytes": size,
			"has_multiple_files": multi, "summary": "Do servidor: " + name, "missing_from_fs": false,
			"path_cover_large": fmt.Sprintf("/assets/romm/resources/roms/1/%d/cover/big.png?ts=2026-10-31 22:24:56", id),
			"metadatum":        map[string]any{"genres": []string{"Plataforma"}, "first_release_date": int64(721008000000), "player_count": "1-2"},
			"files":            files,
		}
	}
	spyro := []map[string]any{
		{"id": 21, "file_name": "Spyro the Dragon (USA).cue", "file_path": "roms/psx/Spyro the Dragon (USA)", "file_size_bytes": len(fakeRomFiles[21])},
		{"id": 22, "file_name": "Spyro the Dragon (USA).bin", "file_path": "roms/psx/Spyro the Dragon (USA)", "file_size_bytes": len(fakeRomFiles[22])},
	}
	return []map[string]any{
		rom(1, "Donkey Kong Country", "Donkey Kong Country (U).zip", "snes", int64(len(fakeRomFiles[1])), false, nil),
		rom(2, "Spyro the Dragon", "Spyro the Dragon (USA)", "psx", int64(len(fakeRomFiles[21])+len(fakeRomFiles[22])), true, spyro),
		rom(3, "Super Mario World", "Super Mario World (U) [!].zip", "snes", int64(len(fakeRomFiles[3])), false, nil),
		rom(4, "Celeste Classic", "celeste.p8", "pico", 100, false, nil), // console que o app não tem
	}
}

// NewFakeRomM monta o servidor falso vazio de saves.
func NewFakeRomM() *FakeRomM {
	f := &FakeRomM{assets: map[string][]fakeAsset{}, nextID: 1, files: maps.Clone(fakeRomFiles)}
	if ms, err := strconv.Atoi(os.Getenv("ARKFLIX_FAKE_ROMM_SLOW")); err == nil && ms > 0 {
		f.Slow = time.Duration(ms) * time.Millisecond
	}
	return f
}

// Put grava um save/state "do servidor" (como se outro aparelho tivesse enviado).
func (f *FakeRomM) Put(kind string, romID int, name string, data []byte, updated time.Time) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.put(kind, romID, name, "snes9x", data, updated)
}

func (f *FakeRomM) put(kind string, romID int, name, emulator string, data []byte, updated time.Time) fakeAsset {
	list := f.assets[kind]
	for i, a := range list {
		if a.RomID == romID && a.FileName == name {
			list[i].data, list[i].Size, list[i].Updated = data, int64(len(data)), updated
			return list[i]
		}
	}
	a := fakeAsset{ID: f.nextID, RomID: romID, FileName: name, Emulator: emulator, Size: int64(len(data)), Updated: updated, data: data}
	f.nextID++
	f.assets[kind] = append(list, a)
	return a
}

// Asset devolve o conteúdo guardado (nil se não existir).
func (f *FakeRomM) Asset(kind string, romID int, name string) []byte {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, a := range f.assets[kind] {
		if a.RomID == romID && a.FileName == name {
			return a.data
		}
	}
	return nil
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(v)
}

func (f *FakeRomM) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	p := r.URL.Path
	switch {
	case p == "/api/token" && r.Method == http.MethodPost:
		r.ParseForm()
		if r.Form.Get("username") != "stanley" || r.Form.Get("password") != "senha123" {
			http.Error(w, `{"detail":"Invalid credentials"}`, http.StatusUnauthorized)
			return
		}
		writeJSON(w, map[string]any{"access_token": fakeAccess, "token_type": "bearer", "expires": 1800})
		return
	case strings.HasPrefix(p, "/assets/") && strings.HasSuffix(p, "/cover/big.png"):
		w.Header().Set("Content-Type", "image/png")
		w.Write(CoverPNG(color.RGBA{R: 200, G: 120, B: 20, A: 255}))
		return
	}
	auth := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
	if auth != fakeDevice && auth != fakeAccess {
		http.Error(w, `{"detail":"Not authenticated"}`, http.StatusUnauthorized)
		return
	}
	switch {
	case p == "/api/client-tokens" && r.Method == http.MethodPost:
		var body struct {
			Name string `json:"name"`
		}
		json.NewDecoder(r.Body).Decode(&body)
		f.mu.Lock()
		f.tokens = append(f.tokens, body.Name)
		f.mu.Unlock()
		writeJSON(w, map[string]any{"id": 7, "name": body.Name, "raw_token": fakeDevice})
	case p == "/api/users/me":
		writeJSON(w, map[string]any{"id": 1, "username": "stanley"})
	case p == "/api/roms":
		offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
		all := f.roms()
		items := all[min(offset, len(all)):]
		writeJSON(w, map[string]any{"items": items, "total": len(all)})
	case strings.HasPrefix(p, "/api/roms/"):
		f.serveRom(w, r, strings.Split(strings.TrimPrefix(p, "/api/roms/"), "/"))
	case p == "/api/platforms":
		out := []map[string]any{}
		for i, slug := range fakePlatforms {
			out = append(out, map[string]any{"id": i + 1, "fs_slug": slug})
		}
		writeJSON(w, out)
	case p == "/api/firmware":
		f.mu.Lock()
		out := []map[string]any{}
		for i, fw := range f.firmware {
			out = append(out, map[string]any{"id": i + 1, "platform_id": slices.Index(fakePlatforms, fw.platform) + 1,
				"file_name": fw.name, "file_size_bytes": len(fw.data), "md5_hash": ""})
		}
		f.mu.Unlock()
		writeJSON(w, out)
	case strings.HasPrefix(p, "/api/firmware/"):
		parts := strings.Split(strings.TrimPrefix(p, "/api/firmware/"), "/")
		id, _ := strconv.Atoi(parts[0])
		f.mu.Lock()
		defer f.mu.Unlock()
		if len(parts) == 3 && parts[1] == "content" && id >= 1 && id <= len(f.firmware) && f.firmware[id-1].name == parts[2] {
			w.Write(f.firmware[id-1].data)
			return
		}
		http.NotFound(w, r)
	case p == "/api/saves" || p == "/api/states":
		f.serveAssets(w, r, strings.TrimPrefix(p, "/api/"))
	case strings.HasSuffix(p, "/content") && (strings.HasPrefix(p, "/api/saves/") || strings.HasPrefix(p, "/api/states/")):
		parts := strings.Split(strings.TrimPrefix(p, "/api/"), "/")
		id, _ := strconv.Atoi(parts[1])
		f.mu.Lock()
		defer f.mu.Unlock()
		for _, a := range f.assets[parts[0]] {
			if a.ID == id {
				w.Write(a.data)
				return
			}
		}
		http.NotFound(w, r)
	default:
		http.NotFound(w, r)
	}
}

func (f *FakeRomM) serveRom(w http.ResponseWriter, r *http.Request, parts []string) {
	id, _ := strconv.Atoi(parts[0])
	if len(parts) == 1 {
		for _, rom := range f.roms() {
			if rom["id"] == id {
				writeJSON(w, rom)
				return
			}
		}
	}
	// /api/roms/<id>/content/<nome> e /api/roms/<arquivo>/files/content/<nome>
	if len(parts) >= 3 && (parts[1] == "content" || parts[1] == "files") {
		f.mu.Lock()
		data, ok := f.files[id]
		if ok && r.Header.Get("Range") != "" {
			f.ranges = append(f.ranges, r.Header.Get("Range"))
		}
		slow := f.Slow
		f.mu.Unlock()
		if ok {
			var out http.ResponseWriter = w
			if slow > 0 {
				out = slowWriter{w, r, slow}
			}
			http.ServeContent(out, r, parts[len(parts)-1], time.Time{}, bytes.NewReader(data))
			return
		}
	}
	http.NotFound(w, r)
}

// slowWriter manda o corpo em pedaços de 16 bytes, com uma pausa entre eles.
type slowWriter struct {
	http.ResponseWriter
	r     *http.Request
	pause time.Duration
}

func (w slowWriter) Write(p []byte) (int, error) {
	n := 0
	for len(p) > 0 {
		k := min(16, len(p))
		m, err := w.ResponseWriter.Write(p[:k])
		n += m
		if err != nil {
			return n, err
		}
		if fl, ok := w.ResponseWriter.(http.Flusher); ok {
			fl.Flush()
		}
		p = p[k:]
		select {
		case <-time.After(w.pause):
		case <-w.r.Context().Done():
			return n, w.r.Context().Err()
		}
	}
	return n, nil
}

func (f *FakeRomM) serveAssets(w http.ResponseWriter, r *http.Request, kind string) {
	romID, _ := strconv.Atoi(r.URL.Query().Get("rom_id"))
	if r.Method == http.MethodPost {
		field := map[string]string{"saves": "saveFile", "states": "stateFile"}[kind]
		file, hdr, err := r.FormFile(field)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		data, _ := io.ReadAll(file)
		f.mu.Lock()
		a := f.put(kind, romID, hdr.Filename, r.URL.Query().Get("emulator"), data, time.Now().UTC())
		f.uploads = append(f.uploads, fmt.Sprintf("%s:%d:%s", kind, romID, hdr.Filename))
		f.mu.Unlock()
		writeJSON(w, a)
		return
	}
	f.mu.Lock()
	out := []fakeAsset{}
	for _, a := range f.assets[kind] {
		if a.RomID == romID {
			out = append(out, a)
		}
	}
	f.mu.Unlock()
	writeJSON(w, out)
}

// BIOS guardada no RomM falso (console › Firmware).
type fakeFirmware struct {
	platform, name string
	data           []byte
}

var fakePlatforms = []string{"genesis", "n64", "nds", "psx", "arcade", "gbc", "nes", "snes", "psp", "ps2"}

// PutFirmware guarda uma BIOS no servidor falso.
func (f *FakeRomM) PutFirmware(platform, name string, data []byte) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.firmware = append(f.firmware, fakeFirmware{platform, name, data})
}
