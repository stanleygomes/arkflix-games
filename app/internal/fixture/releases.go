package fixture

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

// FakeReleases imita o GitHub Releases do app: /repos/<repo>/releases/latest
// e os anexos em /download/<nome> (cada um com o .sha256). Tag vazia = v99.0.0.
type FakeReleases struct {
	Tag   string
	Notes string
	// BadSum: o .sha256 não bate com o arquivo (teste de rejeição).
	BadSum bool
	// Prefix: onde o handler está montado (http.StripPrefix), pros links.
	Prefix string
}

func (f *FakeReleases) tag() string {
	if f.Tag == "" {
		return "v99.0.0"
	}
	return f.Tag
}

// files: os instaladores sem versão no nome, como o bin/package.sh anexa.
func (f *FakeReleases) files() map[string][]byte {
	return map[string][]byte{
		"arkflix-games-linux-x86_64.tar.gz":      FakeTarball("#!/bin/sh\necho " + f.tag() + "\n"),
		"arkflix-games-windows-x86_64-setup.exe": []byte("setup falso " + f.tag()),
		"arkflix-games-macos-universal.dmg":      []byte("dmg falso " + f.tag()),
	}
}

// FakeTarball: tar.gz no layout da release (prefix/bin/arkflix-games).
func FakeTarball(content string) []byte {
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	tw.WriteHeader(&tar.Header{Name: "prefix/bin/arkflix-games", Mode: 0o755, Size: int64(len(content)), Typeflag: tar.TypeReg})
	tw.Write([]byte(content))
	tw.Close()
	gz.Close()
	return buf.Bytes()
}

func (f *FakeReleases) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	files := f.files()
	if strings.HasSuffix(r.URL.Path, "/releases/latest") {
		f.latest(w, r, files)
		return
	}
	name, ok := strings.CutPrefix(r.URL.Path, "/download/")
	if !ok {
		http.NotFound(w, r)
		return
	}
	if base, isSum := strings.CutSuffix(name, ".sha256"); isSum {
		data, found := files[base]
		if !found {
			http.NotFound(w, r)
			return
		}
		sum := sha256.Sum256(data)
		if f.BadSum {
			sum = sha256.Sum256(append(data, 'x'))
		}
		fmt.Fprintf(w, "%s  %s\n", hex.EncodeToString(sum[:]), base)
		return
	}
	data, found := files[name]
	if !found {
		http.NotFound(w, r)
		return
	}
	w.Write(data)
}

func (f *FakeReleases) latest(w http.ResponseWriter, r *http.Request, files map[string][]byte) {
	base := "http://" + r.Host + f.Prefix
	type asset struct {
		Name string `json:"name"`
		Size int    `json:"size"`
		URL  string `json:"browser_download_url"`
	}
	var assets []asset
	for name, data := range files {
		assets = append(assets,
			asset{Name: name, Size: len(data), URL: base + "/download/" + name},
			asset{Name: name + ".sha256", Size: 100, URL: base + "/download/" + name + ".sha256"})
	}
	notes := f.Notes
	if notes == "" {
		notes = "## Novidades\n\n- atualização falsa pra teste\n"
	}
	json.NewEncoder(w).Encode(map[string]any{
		"tag_name": f.tag(), "body": notes, "html_url": base + "/releases/tag/" + f.tag(), "assets": assets,
	})
}
