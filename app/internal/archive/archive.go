// Package archive inspeciona e extrai pacotes de ROM. Zip é nativo; 7z e rar
// usam o binário 7z, se instalado.
package archive

import (
	"archive/zip"
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Kind é o tipo de pacote detectado pelo conteúdo (não pela extensão).
type Kind string

const (
	None     Kind = ""
	Zip      Kind = "zip"
	SevenZip Kind = "7z"
	Rar      Kind = "rar"
)

// Detect olha os primeiros bytes do arquivo.
func Detect(path string) (Kind, error) {
	f, err := os.Open(path)
	if err != nil {
		return None, err
	}
	defer f.Close()
	head := make([]byte, 8)
	n, _ := io.ReadFull(f, head)
	head = head[:n]
	switch {
	case bytes.HasPrefix(head, []byte("PK\x03\x04")), bytes.HasPrefix(head, []byte("PK\x05\x06")):
		return Zip, nil
	case bytes.HasPrefix(head, []byte("7z\xbc\xaf\x27\x1c")):
		return SevenZip, nil
	case bytes.HasPrefix(head, []byte("Rar!\x1a\x07")):
		return Rar, nil
	}
	return None, nil
}

// Entry é um arquivo dentro do pacote.
type Entry struct {
	Name string `json:"name"`
	Size int64  `json:"size"`
}

// List lista os arquivos do pacote (sem diretórios) e confere a integridade do zip.
func List(path string) ([]Entry, error) {
	kind, err := Detect(path)
	if err != nil {
		return nil, err
	}
	switch kind {
	case Zip:
		z, err := zip.OpenReader(path)
		if err != nil {
			return nil, fmt.Errorf("zip corrompido: %w", err)
		}
		defer z.Close()
		var out []Entry
		for _, f := range z.File {
			if f.FileInfo().IsDir() {
				continue
			}
			out = append(out, Entry{Name: f.Name, Size: int64(f.UncompressedSize64)})
		}
		return out, nil
	case SevenZip, Rar:
		return list7z(path)
	}
	return nil, errors.New("não é um pacote (zip, 7z ou rar)")
}

func list7z(path string) ([]Entry, error) {
	bin, err := exec.LookPath("7z")
	if err != nil {
		return nil, errors.New("pacote 7z/rar precisa do 7z instalado (apt install 7zip)")
	}
	out, err := exec.Command(bin, "l", "-ba", "-slt", path).Output()
	if err != nil {
		return nil, fmt.Errorf("pacote corrompido: %w", err)
	}
	var entries []Entry
	var cur Entry
	isDir := false
	flush := func() {
		if cur.Name != "" && !isDir {
			entries = append(entries, cur)
		}
		cur, isDir = Entry{}, false
	}
	for _, line := range strings.Split(string(out), "\n") {
		k, v, ok := strings.Cut(strings.TrimSpace(line), " = ")
		switch {
		case !ok && strings.TrimSpace(line) == "":
			flush()
		case k == "Path":
			flush()
			cur.Name = v
		case k == "Size":
			fmt.Sscan(v, &cur.Size)
		case k == "Attributes":
			isDir = strings.HasPrefix(v, "D")
		}
	}
	flush()
	return entries, nil
}

// Extract extrai tudo em dest, recusando caminhos que escapem da pasta.
func Extract(path, dest string) error {
	kind, err := Detect(path)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dest, 0o755); err != nil {
		return err
	}
	switch kind {
	case Zip:
		return extractZip(path, dest)
	case SevenZip, Rar:
		bin, err := exec.LookPath("7z")
		if err != nil {
			return errors.New("pacote 7z/rar precisa do 7z instalado (apt install 7zip)")
		}
		// lista antes: 7z não tem trava contra caminhos que escapam
		entries, err := list7z(path)
		if err != nil {
			return err
		}
		for _, e := range entries {
			if _, err := safeJoin(dest, e.Name); err != nil {
				return err
			}
		}
		out, err := exec.Command(bin, "x", "-y", "-o"+dest, path).CombinedOutput()
		if err != nil {
			return fmt.Errorf("7z falhou: %s", lastLine(out))
		}
		return nil
	}
	return errors.New("não é um pacote (zip, 7z ou rar)")
}

func extractZip(path, dest string) error {
	z, err := zip.OpenReader(path)
	if err != nil {
		return fmt.Errorf("zip corrompido: %w", err)
	}
	defer z.Close()
	for _, f := range z.File {
		target, err := safeJoin(dest, f.Name)
		if err != nil {
			return err
		}
		if f.FileInfo().IsDir() {
			if err := os.MkdirAll(target, 0o755); err != nil {
				return err
			}
			continue
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		if err := copyZipFile(f, target); err != nil {
			return err
		}
	}
	return nil
}

func copyZipFile(f *zip.File, target string) error {
	src, err := f.Open()
	if err != nil {
		return err
	}
	defer src.Close()
	dst, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		return err
	}
	if _, err := io.Copy(dst, src); err != nil {
		dst.Close()
		return fmt.Errorf("zip corrompido em %s: %w", f.Name, err)
	}
	return dst.Close()
}

func safeJoin(dest, name string) (string, error) {
	target := filepath.Join(dest, filepath.FromSlash(name))
	rel, err := filepath.Rel(dest, target)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) || filepath.IsAbs(name) {
		return "", fmt.Errorf("caminho inseguro no pacote: %s", name)
	}
	return target, nil
}

func lastLine(b []byte) string {
	lines := strings.Split(strings.TrimSpace(string(b)), "\n")
	return lines[len(lines)-1]
}
