package updater

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// API e Repo padrão: a release mais nova do próprio app.
const (
	API  = "https://api.github.com"
	Repo = "stanleygomes/arkflix-games"
)

// Asset é um arquivo anexado na release.
type Asset struct {
	Name string `json:"name"`
	Size int64  `json:"size"`
	URL  string `json:"browser_download_url"`
}

// Release é o que importa de /releases/latest.
type Release struct {
	Tag     string  `json:"tag_name"`
	Body    string  `json:"body"`
	HTMLURL string  `json:"html_url"`
	Assets  []Asset `json:"assets"`
}

// Find acha o anexo pelo nome.
func (r Release) Find(name string) (Asset, bool) {
	for _, a := range r.Assets {
		if a.Name == name {
			return a, true
		}
	}
	return Asset{}, false
}

// AssetName: o instalador sem versão no nome que o CI (bin/package.sh) anexa
// pra cada sistema. "" = sistema sem instalador.
func AssetName(goos, goarch string) string {
	switch {
	case goos == "linux" && goarch == "amd64":
		return "arkflix-games-linux-x86_64.tar.gz"
	case goos == "windows" && goarch == "amd64":
		return "arkflix-games-windows-x86_64-setup.exe"
	case goos == "darwin":
		return "arkflix-games-macos-universal.dmg"
	}
	return ""
}

// Client fala com a API do GitHub. Campos vazios = padrão.
type Client struct {
	API  string // base da API (testes: servidor falso)
	Repo string
	HTTP *http.Client
}

func (c Client) client() *http.Client {
	if c.HTTP != nil {
		return c.HTTP
	}
	return http.DefaultClient
}

func (c Client) get(ctx context.Context, url string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	// a API do GitHub recusa pedido sem User-Agent
	req.Header.Set("User-Agent", "arkflix-games-updater")
	resp, err := c.client().Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, fmt.Errorf("%s respondeu %s", url, resp.Status)
	}
	return resp, nil
}

// Latest busca a release mais nova (o GitHub já tira pré-release e rascunho).
func (c Client) Latest(ctx context.Context) (Release, error) {
	api, repo := c.API, c.Repo
	if api == "" {
		api = API
	}
	if repo == "" {
		repo = Repo
	}
	resp, err := c.get(ctx, strings.TrimRight(api, "/")+"/repos/"+repo+"/releases/latest")
	if err != nil {
		return Release{}, err
	}
	defer resp.Body.Close()
	var rel Release
	if err := json.NewDecoder(io.LimitReader(resp.Body, 4<<20)).Decode(&rel); err != nil {
		return Release{}, fmt.Errorf("resposta da release ilegível: %w", err)
	}
	if rel.Tag == "" {
		return Release{}, errors.New("release sem tag")
	}
	rel.Body = strings.TrimSpace(rel.Body)
	return rel, nil
}
