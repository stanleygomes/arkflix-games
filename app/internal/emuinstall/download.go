package emuinstall

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// Limites de tamanho: o RetroArch do Windows tem ~200 MB compactado e ~550 MB
// aberto; os cores, poucas dezenas de MB. Acima disso é erro, não download.
const (
	maxArchive  = 1 << 30   // 1 GiB baixado
	maxCoreZip  = 256 << 20 // 256 MiB por core
	maxDownload = 20 * time.Minute
)

// download baixa url pra um arquivo temporário em dir, avisando o progresso.
// Quem chama apaga o arquivo.
func (in *Installer) download(ctx context.Context, url, dir string, limit int64, progress func(float64)) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, maxDownload)
	defer cancel()
	resp, err := in.get(ctx, url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.ContentLength > limit {
		return "", fmt.Errorf("arquivo grande demais (%d MB): %s", resp.ContentLength>>20, url)
	}
	f, err := os.CreateTemp(dir, ".arkflix-download-*")
	if err != nil {
		return "", err
	}
	w := &counter{w: f, total: resp.ContentLength, progress: progress}
	n, err := io.Copy(w, io.LimitReader(resp.Body, limit+1))
	if cerr := f.Close(); err == nil {
		err = cerr
	}
	if err == nil && n > limit {
		err = fmt.Errorf("arquivo grande demais: %s", url)
	}
	if err == nil && resp.ContentLength > 0 && n != resp.ContentLength {
		err = fmt.Errorf("download incompleto: %s", url)
	}
	if err != nil {
		os.Remove(f.Name())
		return "", err
	}
	return f.Name(), nil
}

// get faz o GET e confere o status.
func (in *Installer) get(ctx context.Context, url string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "arkflix-games")
	resp, err := in.HTTP.Do(req)
	if err != nil {
		return nil, fmt.Errorf("sem conexão com %s: %w", req.URL.Host, err)
	}
	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, fmt.Errorf("%s respondeu %d em %s", req.URL.Host, resp.StatusCode, url)
	}
	return resp, nil
}

// counter repassa a escrita e avisa a fração já baixada.
type counter struct {
	w        io.Writer
	n, total int64
	progress func(float64)
}

func (c *counter) Write(p []byte) (int, error) {
	n, err := c.w.Write(p)
	c.n += int64(n)
	if c.total > 0 && c.progress != nil {
		c.progress(float64(c.n) / float64(c.total))
	}
	return n, err
}

var stableRe = regexp.MustCompile(`/stable/(\d+)\.(\d+)\.(\d+)/`)

// retroArchVersion: a estável mais nova do buildbot (ou a fixa, se a lista falhar).
func (in *Installer) retroArchVersion(ctx context.Context) string {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	resp, err := in.get(ctx, in.Buildbot+"/stable/")
	if err != nil {
		return RetroArchStable
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	best, bestKey := RetroArchStable, versionKey(RetroArchStable)
	for _, m := range stableRe.FindAllStringSubmatch(string(body), -1) {
		v := m[1] + "." + m[2] + "." + m[3]
		if k := versionKey(v); k > bestKey {
			best, bestKey = v, k
		}
	}
	return best
}

// versionKey: "1.22.2" vira um número comparável.
func versionKey(v string) int64 {
	var key int64
	for _, part := range strings.Split(v, ".") {
		n, _ := strconv.ParseInt(part, 10, 64)
		key = key*10000 + min(n, 9999)
	}
	return key
}

type release struct {
	Tag    string `json:"tag_name"`
	Assets []struct {
		Name string `json:"name"`
		URL  string `json:"browser_download_url"`
	} `json:"assets"`
}

// githubAsset: link do arquivo da última versão cujo nome termina em suffix.
func (in *Installer) githubAsset(ctx context.Context, repo, suffix string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	resp, err := in.get(ctx, in.GitHub+"/repos/"+repo+"/releases/latest")
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	var rel release
	if err := json.NewDecoder(io.LimitReader(resp.Body, 4<<20)).Decode(&rel); err != nil {
		return "", fmt.Errorf("resposta estranha do GitHub: %w", err)
	}
	for _, a := range rel.Assets {
		if strings.HasSuffix(a.Name, suffix) {
			return a.URL, nil
		}
	}
	return "", fmt.Errorf("a versão %s de %s não tem o arquivo *%s", rel.Tag, repo, suffix)
}

// fetchCore baixa o zip do core e põe o arquivo no lugar (troca atômica).
func (in *Installer) fetchCore(ctx context.Context, url, name, dst string, progress func(float64)) error {
	dir := filepath.Dir(dst)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	zipFile, err := in.download(ctx, url, dir, maxCoreZip, progress)
	if err != nil {
		return err
	}
	defer os.Remove(zipFile)
	ok, err := extractOne(zipFile, name, dst)
	if err == nil && !ok {
		err = errors.New(name + " não está no zip")
	}
	return err
}
