// Package romm fala com um servidor RomM (https://romm.app): login, catálogo de
// jogos, download de ROMs e saves/states do usuário. A senha só é usada no
// login: dela sai um token de dispositivo do RomM, que é o que fica guardado.
package romm

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"
)

// Scopes do token do app: ler o catálogo e a BIOS (firmware) e ler/gravar
// saves e states.
var Scopes = []string{"me.read", "roms.read", "platforms.read", "assets.read", "assets.write", "firmware.read"}

var (
	ErrAuth = errors.New("o servidor recusou o acesso: conecte de novo nas configurações")
	ErrHost = errors.New("endereço do servidor inválido")
)

// Account é o que fica guardado depois do login (nunca a senha).
type Account struct {
	Base    string `json:"base"`
	User    string `json:"user"`
	Token   string `json:"token"`
	TokenID int    `json:"tokenId"`
}

// Rom é um jogo do servidor.
type Rom struct {
	ID       int      `json:"id"`
	Name     string   `json:"name"`
	FsName   string   `json:"fsName"`   // nome no disco (arquivo ou pasta)
	Platform string   `json:"platform"` // pasta do console no RomM (snes, psx...)
	Size     int64    `json:"size"`
	Multi    bool     `json:"multi"` // pasta com vários arquivos (discos)
	Summary  string   `json:"summary"`
	Year     int      `json:"year"`
	Genres   []string `json:"genres"`
	Players  string   `json:"players"`
	Cover    string   `json:"cover"` // caminho no servidor (/assets/...)
	Files    []File   `json:"files,omitempty"`
}

// File é um arquivo de um jogo com vários arquivos.
type File struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
	Path string `json:"path"` // pasta no servidor (roms/psx/<jogo>/...)
	Size int64  `json:"size"`
}

// Asset é um save ou state guardado no servidor.
type Asset struct {
	ID       int       `json:"id"`
	FileName string    `json:"fileName"`
	Emulator string    `json:"emulator"`
	Size     int64     `json:"size"`
	Updated  time.Time `json:"updated"`
}

// Kind: "saves" ou "states".
type Kind string

const (
	Saves  Kind = "saves"
	States Kind = "states"
)

// NormalizeBase aceita "servidor.com", "https://servidor.com/" ou um caminho
// (proxy). HTTP puro só pra localhost: o token não viaja aberto na internet.
func NormalizeBase(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", ErrHost
	}
	if !strings.Contains(raw, "://") {
		raw = "https://" + raw
	}
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" || (u.Scheme != "https" && u.Scheme != "http") {
		return "", ErrHost
	}
	if u.Scheme == "http" {
		host := u.Hostname()
		ip := net.ParseIP(host)
		if host != "localhost" && (ip == nil || !ip.IsLoopback()) {
			return "", errors.New("use https (http só pra servidor nesta máquina)")
		}
	}
	u.RawQuery, u.Fragment = "", ""
	return strings.TrimRight(u.String(), "/"), nil
}

// Client chama a API com o token do app.
type Client struct {
	Base  string
	Token string
	HTTP  *http.Client
	// Limit: velocidade máxima dos downloads em bytes/s (0 ou nil = sem limite).
	Limit func() int64
}

func (c *Client) http() *http.Client {
	if c.HTTP != nil {
		return c.HTTP
	}
	return &http.Client{Timeout: 0} // download grande: o prazo vem do ctx
}

func (c *Client) do(ctx context.Context, method, p string, body io.Reader, ctype string) (*http.Response, error) {
	var hdr http.Header
	if ctype != "" {
		hdr = http.Header{"Content-Type": {ctype}}
	}
	return c.doHeader(ctx, method, p, body, hdr)
}

func (c *Client) doHeader(ctx context.Context, method, p string, body io.Reader, hdr http.Header) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, method, c.Base+p, body)
	if err != nil {
		return nil, err
	}
	for k, v := range hdr {
		req.Header[k] = v
	}
	if c.Token != "" {
		req.Header.Set("Authorization", "Bearer "+c.Token)
	}
	resp, err := c.http().Do(req)
	if err != nil {
		return nil, fmt.Errorf("servidor fora do ar? %w", err)
	}
	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		resp.Body.Close()
		return nil, ErrAuth
	}
	if resp.StatusCode >= 300 {
		msg, _ := io.ReadAll(io.LimitReader(resp.Body, 300))
		resp.Body.Close()
		return nil, fmt.Errorf("servidor respondeu %d: %s", resp.StatusCode, strings.TrimSpace(string(msg)))
	}
	return resp, nil
}

func (c *Client) getJSON(ctx context.Context, p string, v any) error {
	resp, err := c.do(ctx, http.MethodGet, p, nil, "")
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return json.NewDecoder(resp.Body).Decode(v)
}

// Login troca usuário e senha por um token de dispositivo (sem validade) com
// só os escopos do app. A senha não sai daqui.
func Login(ctx context.Context, base, user, pass, device string) (Account, error) {
	base, err := NormalizeBase(base)
	if err != nil {
		return Account{}, err
	}
	form := url.Values{"grant_type": {"password"}, "username": {user}, "password": {pass},
		"scope": {strings.Join(append([]string{"me.write"}, Scopes...), " ")}}
	c := &Client{Base: base, HTTP: &http.Client{Timeout: 30 * time.Second}}
	resp, err := c.do(ctx, http.MethodPost, "/api/token", strings.NewReader(form.Encode()), "application/x-www-form-urlencoded")
	if err != nil {
		if errors.Is(err, ErrAuth) {
			return Account{}, errors.New("usuário ou senha errados")
		}
		return Account{}, err
	}
	var tok struct {
		Access string `json:"access_token"`
	}
	err = json.NewDecoder(resp.Body).Decode(&tok)
	resp.Body.Close()
	if err != nil || tok.Access == "" {
		return Account{}, errors.New("isso não parece um servidor RomM")
	}
	c.Token = tok.Access
	body, _ := json.Marshal(map[string]any{"name": device, "scopes": Scopes})
	resp, err = c.do(ctx, http.MethodPost, "/api/client-tokens", bytes.NewReader(body), "application/json")
	if err != nil {
		return Account{}, fmt.Errorf("criar o token do app: %w", err)
	}
	defer resp.Body.Close()
	var ct struct {
		ID  int    `json:"id"`
		Raw string `json:"raw_token"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&ct); err != nil || ct.Raw == "" {
		return Account{}, errors.New("o servidor não devolveu o token do app (RomM 4.4 ou mais novo?)")
	}
	return Account{Base: base, User: user, Token: ct.Raw, TokenID: ct.ID}, nil
}

// Me confere o token e devolve o nome do usuário.
func (c *Client) Me(ctx context.Context) (string, error) {
	var me struct {
		Username string `json:"username"`
	}
	err := c.getJSON(ctx, "/api/users/me", &me)
	return me.Username, err
}

type apiRom struct {
	ID            int    `json:"id"`
	Name          string `json:"name"`
	FsName        string `json:"fs_name"`
	Platform      string `json:"platform_fs_slug"`
	Size          int64  `json:"fs_size_bytes"`
	Multi         bool   `json:"has_multiple_files"`
	Summary       string `json:"summary"`
	Cover         string `json:"path_cover_large"`
	MissingFromFS bool   `json:"missing_from_fs"`
	Metadatum     struct {
		Genres  []string `json:"genres"`
		Release int64    `json:"first_release_date"` // ms
		Players string   `json:"player_count"`
	} `json:"metadatum"`
	Files []struct {
		ID   int    `json:"id"`
		Name string `json:"file_name"`
		Path string `json:"file_path"`
		Size int64  `json:"file_size_bytes"`
	} `json:"files"`
}

func (r apiRom) rom() Rom {
	out := Rom{ID: r.ID, Name: r.Name, FsName: r.FsName, Platform: r.Platform, Size: r.Size, Multi: r.Multi,
		Summary: r.Summary, Genres: r.Metadatum.Genres, Players: r.Metadatum.Players, Cover: r.Cover}
	if r.Metadatum.Release > 0 {
		out.Year = time.UnixMilli(r.Metadatum.Release).UTC().Year()
	}
	for _, f := range r.Files {
		out.Files = append(out.Files, File{ID: f.ID, Name: f.Name, Path: f.Path, Size: f.Size})
	}
	return out
}

// Roms lista o catálogo inteiro (paginado).
func (c *Client) Roms(ctx context.Context) ([]Rom, error) {
	var out []Rom
	for offset := 0; ; {
		var page struct {
			Items []apiRom `json:"items"`
			Total int      `json:"total"`
		}
		if err := c.getJSON(ctx, fmt.Sprintf("/api/roms?limit=200&offset=%d&with_char_index=false&with_filter_values=false", offset), &page); err != nil {
			return nil, err
		}
		for _, r := range page.Items {
			if !r.MissingFromFS {
				out = append(out, r.rom())
			}
		}
		offset += len(page.Items)
		if len(page.Items) == 0 || offset >= page.Total {
			return out, nil
		}
	}
}

// Rom devolve um jogo com a lista de arquivos.
func (c *Client) Rom(ctx context.Context, id int) (Rom, error) {
	var r apiRom
	err := c.getJSON(ctx, fmt.Sprintf("/api/roms/%d", id), &r)
	return r.rom(), err
}

// PartPath é onde fica o download pela metade do jogo (arquivo ou pasta).
func PartPath(dir string, r Rom) (string, error) {
	if r.FsName == "" || strings.ContainsAny(r.FsName, `/\`) || r.FsName == "." || r.FsName == ".." {
		return "", fmt.Errorf("nome de jogo inválido: %q", r.FsName)
	}
	return filepath.Join(dir, r.FsName) + ".part", nil
}

// Download baixa o jogo pra dir/<FsName> (arquivo ou pasta). Grava em .part e
// só renomeia no fim: download pela metade nunca vira jogo na biblioteca.
// Se o .part já existe (download pausado), continua de onde parou: arquivo
// único pede o resto com Range; jogo com vários arquivos pula os completos.
// Em erro o .part fica (dá pra continuar); quem desiste apaga (PartPath).
func (c *Client) Download(ctx context.Context, r Rom, dir string, progress func(done, total int64)) (string, error) {
	part, err := PartPath(dir, r)
	if err != nil {
		return "", err
	}
	dest := strings.TrimSuffix(part, ".part")
	if _, err := os.Stat(dest); err == nil {
		return "", fmt.Errorf("já existe %s", dest)
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	var done int64
	count := func(n int64) {
		done += n
		if progress != nil {
			progress(done, r.Size)
		}
	}
	if !r.Multi {
		if st, err := os.Stat(part); err == nil && st.IsDir() {
			os.RemoveAll(part) // sobra de outro formato: recomeça
		}
		if err := c.resume(ctx, fmt.Sprintf("/api/roms/%d/content/%s", r.ID, url.PathEscape(r.FsName)), part, r.Size, count); err != nil {
			return "", err
		}
		return dest, os.Rename(part, dest)
	}
	if st, err := os.Stat(part); err == nil && !st.IsDir() {
		os.Remove(part)
	}
	prefix := path.Join("roms", r.Platform, r.FsName)
	for _, f := range r.Files {
		clean := path.Clean(f.Path)
		rel, inside := strings.CutPrefix(clean, prefix)
		inside = inside && (rel == "" || strings.HasPrefix(rel, "/"))
		target := filepath.Join(part, filepath.FromSlash(rel), f.Name)
		if !inside || f.Name == "" || f.Name == ".." || strings.ContainsAny(f.Name, `/\`) || !strings.HasPrefix(target, part+string(filepath.Separator)) {
			os.RemoveAll(part)
			return "", fmt.Errorf("caminho de arquivo inválido no servidor: %s/%s", f.Path, f.Name)
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return "", err
		}
		if err := c.resume(ctx, fmt.Sprintf("/api/roms/%d/files/content/%s", f.ID, url.PathEscape(f.Name)), target, f.Size, count); err != nil {
			return "", err
		}
	}
	return dest, os.Rename(part, dest)
}

type counter struct {
	w   io.Writer
	add func(int64)
}

func (c counter) Write(p []byte) (int, error) {
	n, err := c.w.Write(p)
	c.add(int64(n))
	return n, err
}

// limited segura a leitura pra não passar de rate() bytes por segundo (lido a
// cada pedaço: mudar o limite vale pro download em andamento).
type limited struct {
	ctx   context.Context
	r     io.Reader
	rate  func() int64
	cur   int64
	start time.Time
	n     int64
}

func (l *limited) Read(p []byte) (int, error) {
	rate := l.rate()
	if rate <= 0 {
		return l.r.Read(p)
	}
	if rate != l.cur {
		l.cur, l.start, l.n = rate, time.Now(), 0
	}
	if chunk := max(rate/8, 1024); int64(len(p)) > chunk {
		p = p[:chunk]
	}
	n, err := l.r.Read(p)
	l.n += int64(n)
	wait := time.Duration(float64(l.n)/float64(rate)*float64(time.Second)) - time.Since(l.start)
	if wait > 0 {
		t := time.NewTimer(wait)
		defer t.Stop()
		select {
		case <-t.C:
		case <-l.ctx.Done():
			return n, l.ctx.Err()
		}
	}
	return n, err
}

func (c *Client) body(ctx context.Context, r io.Reader) io.Reader {
	if c.Limit == nil {
		return r
	}
	return &limited{ctx: ctx, r: r, rate: c.Limit}
}

func (c *Client) fetch(ctx context.Context, p, dest string, add func(int64)) error {
	resp, err := c.do(ctx, http.MethodGet, p, nil, "")
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	f, err := os.Create(dest)
	if err != nil {
		return err
	}
	_, err = io.Copy(counter{f, add}, c.body(ctx, resp.Body))
	if cerr := f.Close(); err == nil {
		err = cerr
	}
	return err
}

// resume baixa p pra dest continuando o que já está lá (want = tamanho
// final, 0 = desconhecido). O RomM serve o arquivo pelo nginx
// (X-Accel-Redirect), que aceita Range; se o servidor ignorar o Range e
// responder 200 com o arquivo inteiro, recomeça do zero.
func (c *Client) resume(ctx context.Context, p, dest string, want int64, add func(int64)) error {
	var have int64
	if st, err := os.Stat(dest); err == nil && st.Mode().IsRegular() {
		have = st.Size()
	}
	if want > 0 && have == want {
		add(have) // já completo (jogo com vários arquivos pausado no seguinte)
		return nil
	}
	if want > 0 && have > want {
		have = 0
	}
	var hdr http.Header
	if have > 0 {
		hdr = http.Header{"Range": {fmt.Sprintf("bytes=%d-", have)}}
	}
	resp, err := c.doHeader(ctx, http.MethodGet, p, nil, hdr)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	flag := os.O_CREATE | os.O_WRONLY | os.O_TRUNC
	if have > 0 && resp.StatusCode == http.StatusPartialContent {
		flag = os.O_CREATE | os.O_WRONLY | os.O_APPEND
		add(have)
	}
	f, err := os.OpenFile(dest, flag, 0o644)
	if err != nil {
		return err
	}
	_, err = io.Copy(counter{f, add}, c.body(ctx, resp.Body))
	if cerr := f.Close(); err == nil {
		err = cerr
	}
	return err
}

// Assets lista os saves ou states do usuário pra um jogo.
func (c *Client) Assets(ctx context.Context, kind Kind, romID int) ([]Asset, error) {
	var raw []struct {
		ID       int       `json:"id"`
		FileName string    `json:"file_name"`
		Emulator string    `json:"emulator"`
		Size     int64     `json:"file_size_bytes"`
		Updated  time.Time `json:"updated_at"`
	}
	if err := c.getJSON(ctx, fmt.Sprintf("/api/%s?rom_id=%d", kind, romID), &raw); err != nil {
		return nil, err
	}
	out := make([]Asset, 0, len(raw))
	for _, a := range raw {
		out = append(out, Asset{ID: a.ID, FileName: a.FileName, Emulator: a.Emulator, Size: a.Size, Updated: a.Updated})
	}
	return out, nil
}

// Upload envia (ou substitui, pelo nome) um save ou state. shot = miniatura opcional.
func (c *Client) Upload(ctx context.Context, kind Kind, romID int, emulator, file, shot string) error {
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	field := map[Kind]string{Saves: "saveFile", States: "stateFile"}[kind]
	for _, part := range []struct{ field, path string }{{field, file}, {"screenshotFile", shot}} {
		if part.path == "" {
			continue
		}
		data, err := os.ReadFile(part.path)
		if err != nil {
			return err
		}
		fw, err := w.CreateFormFile(part.field, filepath.Base(part.path))
		if err != nil {
			return err
		}
		fw.Write(data)
	}
	w.Close()
	q := url.Values{"rom_id": {fmt.Sprint(romID)}, "emulator": {emulator}}
	if kind == Saves {
		q.Set("overwrite", "true")
	}
	resp, err := c.do(ctx, http.MethodPost, fmt.Sprintf("/api/%s?%s", kind, q.Encode()), &buf, w.FormDataContentType())
	if err != nil {
		return err
	}
	resp.Body.Close()
	return nil
}

// Fetch baixa um save ou state pra dest, com a data do servidor.
func (c *Client) Fetch(ctx context.Context, kind Kind, a Asset, dest string) error {
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return err
	}
	part := dest + ".part"
	if err := c.fetch(ctx, fmt.Sprintf("/api/%s/%d/content", kind, a.ID), part, func(int64) {}); err != nil {
		os.Remove(part)
		return err
	}
	if err := os.Rename(part, dest); err != nil {
		return err
	}
	return os.Chtimes(dest, a.Updated, a.Updated)
}

// Cover baixa a capa (caminho do servidor) pra dest.
func (c *Client) Cover(ctx context.Context, p, dest string) error {
	p, _, _ = strings.Cut(p, "?") // ?ts= é só pra cache do navegador
	if !strings.HasPrefix(p, "/assets/") || strings.Contains(p, "..") {
		return errors.New("capa fora do servidor")
	}
	p = (&url.URL{Path: p}).EscapedPath()
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return err
	}
	part := dest + ".part"
	if err := c.fetch(ctx, p, part, func(int64) {}); err != nil {
		os.Remove(part)
		return err
	}
	return os.Rename(part, dest)
}

// Firmware é uma BIOS guardada no RomM (página do console › Firmware).
type Firmware struct {
	ID       int
	FileName string
	Size     int64
	MD5      string
	Platform string // pasta do console no RomM (psx, ps2...)
}

// Firmwares lista as BIOS do servidor, com o console de cada uma.
func (c *Client) Firmwares(ctx context.Context) ([]Firmware, error) {
	var plats []struct {
		ID     int    `json:"id"`
		FsSlug string `json:"fs_slug"`
	}
	if err := c.getJSON(ctx, "/api/platforms", &plats); err != nil {
		return nil, err
	}
	slug := map[int]string{}
	for _, p := range plats {
		slug[p.ID] = p.FsSlug
	}
	var raw []struct {
		ID         int    `json:"id"`
		PlatformID int    `json:"platform_id"`
		FileName   string `json:"file_name"`
		Size       int64  `json:"file_size_bytes"`
		MD5        string `json:"md5_hash"`
	}
	if err := c.getJSON(ctx, "/api/firmware", &raw); err != nil {
		return nil, err
	}
	out := make([]Firmware, 0, len(raw))
	for _, f := range raw {
		out = append(out, Firmware{ID: f.ID, FileName: f.FileName, Size: f.Size, MD5: f.MD5, Platform: slug[f.PlatformID]})
	}
	return out, nil
}

// FetchFirmware baixa a BIOS pra dest (temporário + rename: nunca deixa
// arquivo pela metade com o nome certo).
func (c *Client) FetchFirmware(ctx context.Context, f Firmware, dest string) error {
	if f.FileName == "" || filepath.Base(f.FileName) != f.FileName || f.FileName == ".." {
		return fmt.Errorf("nome de BIOS inválido no servidor: %q", f.FileName)
	}
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return err
	}
	part := dest + ".part"
	if err := c.fetch(ctx, fmt.Sprintf("/api/firmware/%d/content/%s", f.ID, url.PathEscape(f.FileName)), part, func(int64) {}); err != nil {
		os.Remove(part)
		return err
	}
	return os.Rename(part, dest)
}
