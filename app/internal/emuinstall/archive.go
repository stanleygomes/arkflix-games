package emuinstall

import (
	"archive/zip"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/bodgit/sevenzip"
)

// Limites da extração: o RetroArch do Windows tem ~15 mil arquivos e ~550 MB.
// Zip-bomba ou arquivo estranho para aqui.
const (
	maxFiles     = 50_000
	maxExtracted = 4 << 30 // 4 GiB no total
)

// entry é um arquivo de zip ou 7z, do jeito que a extração precisa.
type entry struct {
	name string
	mode fs.FileMode
	size uint64
	open func() (io.ReadCloser, error)
}

// openArchive lê a lista de um .zip ou .7z (pela extensão).
func openArchive(file string) ([]entry, io.Closer, error) {
	if strings.HasSuffix(strings.ToLower(file), ".7z") {
		r, err := sevenzip.OpenReader(file)
		if err != nil {
			return nil, nil, fmt.Errorf("7z inválido: %w", err)
		}
		out := make([]entry, len(r.File))
		for i, f := range r.File {
			out[i] = entry{name: f.Name, mode: f.Mode(), size: f.UncompressedSize, open: f.Open}
		}
		return out, r, nil
	}
	r, err := zip.OpenReader(file)
	if err != nil {
		return nil, nil, fmt.Errorf("zip inválido: %w", err)
	}
	out := make([]entry, len(r.File))
	for i, f := range r.File {
		out[i] = entry{name: f.Name, mode: f.Mode(), size: f.UncompressedSize64, open: f.Open}
	}
	return out, r, nil
}

// clean normaliza o nome (barras do Windows) e recusa o que sai da pasta.
func clean(name string) (string, error) {
	n := strings.ReplaceAll(name, `\`, "/")
	if n == "" || strings.HasPrefix(n, "/") || strings.Contains(n, ":") || !filepath.IsLocal(filepath.FromSlash(n)) {
		return "", fmt.Errorf("caminho perigoso no pacote: %q", name)
	}
	return path.Clean(n), nil
}

// extractTree extrai o pacote em dst por cima do que existe (mantém config e
// saves de instalações portáteis). A pasta raiz é a do executável exe dentro
// do pacote (o RetroArch vem dentro de RetroArch-Win64/).
func extractTree(file, exe, dst string) error {
	entries, closer, err := openArchive(file)
	if err != nil {
		return err
	}
	defer closer.Close()
	root, err := rootOf(entries, exe)
	if err != nil {
		return err
	}
	var total uint64
	count := 0
	for _, e := range entries {
		name, err := clean(e.name)
		if err != nil {
			return err
		}
		rel, ok := strings.CutPrefix(name, root)
		if !ok || rel == "" || rel == "." || e.mode&fs.ModeSymlink != 0 {
			continue // fora da raiz ou link: ignora
		}
		target := filepath.Join(dst, filepath.FromSlash(rel))
		if e.mode.IsDir() {
			if err := os.MkdirAll(target, 0o755); err != nil {
				return err
			}
			continue
		}
		if count++; count > maxFiles {
			return errors.New("pacote com arquivos demais")
		}
		if total += e.size; total > maxExtracted {
			return errors.New("pacote grande demais depois de aberto")
		}
		if err := writeEntry(e, target); err != nil {
			return err
		}
	}
	return nil
}

// rootOf: prefixo (com barra) do executável dentro do pacote.
func rootOf(entries []entry, exe string) (string, error) {
	best := ""
	found := false
	for _, e := range entries {
		name, err := clean(e.name)
		if err != nil {
			return "", err
		}
		if !e.mode.IsDir() && strings.EqualFold(path.Base(name), exe) {
			dir := path.Dir(name) + "/"
			if dir == "./" {
				dir = ""
			}
			if !found || len(dir) < len(best) {
				best, found = dir, true
			}
		}
	}
	if !found {
		return "", fmt.Errorf("%s não está no pacote", exe)
	}
	return best, nil
}

// extractOne põe o arquivo name do zip em dst. false = não achou.
func extractOne(file, name, dst string) (bool, error) {
	entries, closer, err := openArchive(file)
	if err != nil {
		return false, err
	}
	defer closer.Close()
	for _, e := range entries {
		n, err := clean(e.name)
		if err != nil {
			return false, err
		}
		if path.Base(n) == name && e.mode.IsRegular() {
			return true, writeEntry(e, dst)
		}
	}
	return false, nil
}

// writeEntry grava num temporário ao lado e troca (nunca deixa arquivo pela
// metade). Lê no máximo o tamanho declarado: zip mentiroso falha.
func writeEntry(e entry, target string) error {
	if e.size > maxExtracted {
		return errors.New("arquivo grande demais no pacote")
	}
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return err
	}
	rc, err := e.open()
	if err != nil {
		return err
	}
	defer rc.Close()
	tmp, err := os.CreateTemp(filepath.Dir(target), ".arkflix-*")
	if err != nil {
		return err
	}
	n, err := io.Copy(tmp, io.LimitReader(rc, int64(e.size)+1))
	if cerr := tmp.Close(); err == nil {
		err = cerr
	}
	if err == nil && uint64(n) != e.size {
		err = fmt.Errorf("tamanho errado no pacote: %s", e.name)
	}
	mode := fs.FileMode(0o644)
	if e.mode&0o111 != 0 || strings.HasSuffix(strings.ToLower(target), ".exe") {
		mode = 0o755
	}
	if err == nil {
		err = os.Chmod(tmp.Name(), mode)
	}
	if err == nil {
		err = os.Rename(tmp.Name(), target)
	}
	if err != nil {
		os.Remove(tmp.Name())
		if errors.Is(err, fs.ErrPermission) {
			return fmt.Errorf("sem permissão pra escrever em %s (o emulador está aberto?)", filepath.Dir(target))
		}
		return err
	}
	return nil
}
