package updater

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// Sum lê o .sha256 do anexo ("<hash>  <nome>", formato do sha256sum).
func (c Client) Sum(ctx context.Context, url string) (string, error) {
	resp, err := c.get(ctx, url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(io.LimitReader(resp.Body, 4096))
	if err != nil {
		return "", err
	}
	fields := strings.Fields(string(data))
	if len(fields) == 0 {
		return "", errors.New("checksum vazio")
	}
	sum := strings.ToLower(fields[0])
	if b, err := hex.DecodeString(sum); err != nil || len(b) != sha256.Size {
		return "", errors.New("checksum inválido")
	}
	return sum, nil
}

// Download baixa url em dst conferindo o sha256. Só chega em dst o arquivo
// conferido: baixa em .part e renomeia no fim.
func (c Client) Download(ctx context.Context, url, sum, dst string, size int64, progress func(float64)) error {
	resp, err := c.get(ctx, url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if size <= 0 {
		size = resp.ContentLength
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	part := dst + ".part"
	f, err := os.Create(part)
	if err != nil {
		return err
	}
	defer os.Remove(part) // depois do rename não existe mais: sem efeito
	h := sha256.New()
	_, err = io.Copy(io.MultiWriter(f, h), &progressReader{r: resp.Body, size: size, fn: progress})
	if cerr := f.Close(); err == nil {
		err = cerr
	}
	if err != nil {
		return fmt.Errorf("download interrompido: %w", err)
	}
	if got := hex.EncodeToString(h.Sum(nil)); got != sum {
		return fmt.Errorf("checksum não confere (esperado %s, veio %s): arquivo descartado", sum, got)
	}
	return os.Rename(part, dst)
}

// progressReader avisa a fração lida (sem tamanho conhecido, não avisa).
type progressReader struct {
	r    io.Reader
	size int64
	n    int64
	fn   func(float64)
}

func (p *progressReader) Read(b []byte) (int, error) {
	n, err := p.r.Read(b)
	p.n += int64(n)
	if p.fn != nil && p.size > 0 && n > 0 {
		p.fn(min(1, float64(p.n)/float64(p.size)))
	}
	return n, err
}
