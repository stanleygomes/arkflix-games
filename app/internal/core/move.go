package core

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/stanleygomes/arkflix-games/app/internal/catalog"
	"github.com/stanleygomes/arkflix-games/app/internal/library"
	"github.com/stanleygomes/arkflix-games/app/internal/settings"
)

// Mover os jogos de um console pra outra pasta da biblioteca (outro disco).
// O id do jogo é console + nome do arquivo (library.ID), não o caminho: depois
// de mover, favoritos, tempo de jogo, listas, histórico e metadados editados
// continuam valendo. Os saves também (são pelo nome do jogo ou pelo serial).

// Ganchos dos testes (outro disco, espaço livre, cancelar no meio da cópia).
var (
	moveRename   = os.Rename
	moveSameDisk = sameDisk
	moveFree     = freeSpace
	moveChunk    = func(dst string) {}
)

// moveMedia: imagens e vídeos locais do jogo, ao lado dele (<raiz>/<console>/covers...).
var moveMedia = []string{"covers", "screenshots", "manuals", "videos"}

const moveTmpPrefix = ".arkflix-move-" // começa com ponto: a varredura ignora

// MoveTarget é um destino possível pros jogos de um console.
type MoveTarget struct {
	Path     string `json:"path"`
	Label    string `json:"label"`
	Free     int64  `json:"free"`     // livre no disco do destino (-1: não deu pra medir)
	Size     int64  `json:"size"`     // tamanho dos jogos a mover
	Count    int    `json:"count"`    // quantos jogos
	SameDisk bool   `json:"sameDisk"` // mesmo disco: só renomeia, não ocupa espaço
	Fits     bool   `json:"fits"`
}

// libraryDir acha a pasta da biblioteca pelo caminho (só as configuradas valem).
func libraryDir(dirs []settings.LibraryDir, path string) (settings.LibraryDir, bool) {
	if path == "" {
		return settings.LibraryDir{}, false
	}
	clean := filepath.Clean(path)
	for _, d := range dirs {
		if d.Path != "" && filepath.Clean(d.Path) == clean {
			return d, true
		}
	}
	return settings.LibraryDir{}, false
}

// gamesIn: os jogos do console numa pasta da biblioteca (inclusive os que a
// lista esconde por existirem também noutra pasta).
func gamesIn(dir settings.LibraryDir, system string) []library.Game {
	var out []library.Game
	for _, g := range library.Scan([]settings.LibraryDir{dir}) {
		if g.System == system {
			out = append(out, g)
		}
	}
	return out
}

func (s *Service) movePlan(system, from string) (catalog.System, settings.LibraryDir, []library.Game, []MoveTarget, error) {
	if err := s.adult(); err != nil {
		return catalog.System{}, settings.LibraryDir{}, nil, nil, err
	}
	sys, ok := catalog.Find(system)
	if !ok {
		return sys, settings.LibraryDir{}, nil, nil, errors.New("console desconhecido")
	}
	dirs := s.opt.Store.Settings().LibraryDirs
	src, ok := libraryDir(dirs, from)
	if !ok {
		return sys, src, nil, nil, errors.New("a origem não é uma pasta da biblioteca")
	}
	games := gamesIn(src, sys.Slug)
	var size int64
	for _, g := range games {
		size += g.Size
	}
	targets := []MoveTarget{}
	for _, d := range dirs {
		if !d.Writable || d.Path == "" || filepath.Clean(d.Path) == filepath.Clean(src.Path) {
			continue
		}
		t := MoveTarget{Path: d.Path, Label: d.Label, Size: size, Count: len(games), Free: moveFree(d.Path), SameDisk: moveSameDisk(src.Path, d.Path)}
		if t.Label == "" {
			t.Label = filepath.Base(d.Path)
		}
		t.Fits = t.SameDisk || t.Free < 0 || t.Free >= size
		targets = append(targets, t)
	}
	return sys, src, games, targets, nil
}

// MovePlan lista pra onde dá pra mover os jogos do console que estão na pasta
// from: as outras pastas graváveis da biblioteca, com espaço livre e tamanho.
func (s *Service) MovePlan(system, from string) ([]MoveTarget, error) {
	_, _, _, targets, err := s.movePlan(system, from)
	return targets, err
}

// MoveGames move os jogos do console da pasta from pra pasta to (as duas da
// biblioteca; to gravável). Roda como tarefa "move:<console>" na Central, com
// progresso por bytes e cancelável. Jogo com o mesmo nome no destino fica
// onde está (pulado). No mesmo disco só renomeia; noutro, copia, confere o
// tamanho e só então apaga a origem.
func (s *Service) MoveGames(system, from, to string) error {
	sys, src, games, targets, err := s.movePlan(system, from)
	if err != nil {
		return err
	}
	var dst *MoveTarget
	for i := range targets {
		if filepath.Clean(targets[i].Path) == filepath.Clean(to) {
			dst = &targets[i]
		}
	}
	switch {
	case dst == nil:
		return errors.New("o destino tem que ser outra pasta gravável da biblioteca")
	case len(games) == 0:
		return fmt.Errorf("nenhum jogo de %s nessa pasta", sys.Name)
	case !dst.Fits:
		return fmt.Errorf("falta espaço no destino: precisa de %s, tem %s livre", humanBytes(dst.Size), humanBytes(dst.Free))
	case s.Running() != "":
		return errors.New("feche o jogo antes de mover")
	}
	target := dst.Path
	ok := s.runTaskCtx("move:"+sys.Slug, "Movendo jogos de "+sys.Name, func(ctx context.Context, progress func(float64)) (string, error) {
		defer func() {
			s.scan()
			s.opt.Emitter.Emit("library", s.views())
		}()
		total := dst.Size
		var done int64
		last := -1.0
		add := func(n int64) {
			done += n
			if total <= 0 {
				return
			}
			if p := float64(done) / float64(total); p-last >= 0.01 || p >= 1 {
				last = p
				progress(min(p, 1))
			}
		}
		moved, skipped := 0, 0
		for _, g := range games {
			if ctx.Err() != nil {
				return "", ctx.Err()
			}
			out := filepath.Join(target, g.Folder, g.FileName)
			if _, err := os.Lstat(out); err == nil {
				skipped++
				add(g.Size)
				continue
			}
			before := done
			if err := moveEntry(ctx, g.Entry, out, add); err != nil {
				if ctx.Err() != nil {
					return "", ctx.Err()
				}
				return "", fmt.Errorf("%s: %w (%d movido(s) antes)", g.Title, err, moved)
			}
			add(g.Size - (done - before)) // rename não conta bytes
			moved++
			moveSidecars(ctx, g, src.Path, target)
		}
		msg := fmt.Sprintf("%d jogo(s) movido(s) pra %s", moved, dst.Label)
		if skipped > 0 {
			msg += fmt.Sprintf(", %d pulado(s): já existia(m) no destino", skipped)
		}
		return msg, nil
	})
	if !ok {
		return errors.New("os jogos desse console já estão sendo movidos")
	}
	return nil
}

// moveEntry move um arquivo ou pasta: rename; se for outro disco, copia pra um
// temporário ao lado do destino, confere o tamanho, renomeia e só então apaga
// a origem. Cancelado ou com erro: o temporário some e a origem fica.
func moveEntry(ctx context.Context, src, dst string, add func(int64)) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	err := moveRename(src, dst)
	if err == nil || !crossDevice(err) {
		return err
	}
	tmp := filepath.Join(filepath.Dir(dst), moveTmpPrefix+filepath.Base(dst))
	os.RemoveAll(tmp) // sobra de uma vez que caiu no meio
	if err := copyTree(ctx, src, tmp, add); err != nil {
		os.RemoveAll(tmp)
		return err
	}
	if want, got := treeSize(src), treeSize(tmp); want != got {
		os.RemoveAll(tmp)
		return fmt.Errorf("a cópia não confere (%d de %d bytes)", got, want)
	}
	if _, err := os.Lstat(dst); err == nil {
		os.RemoveAll(tmp)
		return errors.New("apareceu um arquivo com o mesmo nome no destino")
	}
	if err := os.Rename(tmp, dst); err != nil {
		os.RemoveAll(tmp)
		return err
	}
	if err := os.RemoveAll(src); err != nil {
		return fmt.Errorf("copiado, mas não deu pra apagar a origem: %w", err)
	}
	return nil
}

// treeSize soma os arquivos comuns (a pasta ou o arquivo); -1 se não deu pra ler.
func treeSize(path string) int64 {
	var total int64
	err := filepath.WalkDir(path, func(_ string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.Type().IsRegular() {
			info, err := d.Info()
			if err != nil {
				return err
			}
			total += info.Size()
		}
		return nil
	})
	if err != nil {
		return -1
	}
	return total
}

func copyTree(ctx context.Context, src, dst string, add func(int64)) error {
	return filepath.WalkDir(src, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		out := filepath.Join(dst, rel)
		switch {
		case d.IsDir():
			return os.MkdirAll(out, 0o755)
		case d.Type()&fs.ModeSymlink != 0:
			link, err := os.Readlink(path)
			if err != nil {
				return err
			}
			return os.Symlink(link, out)
		case d.Type().IsRegular():
			return copyFile(ctx, path, out, add)
		}
		return nil // socket, device: não é jogo
	})
}

func copyFile(ctx context.Context, src, dst string, add func(int64)) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	info, err := in.Stat()
	if err != nil {
		return err
	}
	out, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_EXCL, info.Mode().Perm())
	if err != nil {
		return err
	}
	buf := make([]byte, 4<<20)
	for {
		if err := ctx.Err(); err != nil {
			out.Close()
			return err
		}
		n, rerr := in.Read(buf)
		if n > 0 {
			if _, err := out.Write(buf[:n]); err != nil {
				out.Close()
				return err
			}
			add(int64(n))
			moveChunk(dst)
		}
		if rerr == io.EOF {
			break
		}
		if rerr != nil {
			out.Close()
			return rerr
		}
	}
	if err := out.Sync(); err != nil {
		out.Close()
		return err
	}
	if err := out.Close(); err != nil {
		return err
	}
	os.Chtimes(dst, info.ModTime(), info.ModTime()) // "adicionado em" não muda
	return nil
}

// moveSidecars leva junto capa, print, manual e vídeo locais do jogo. Falha
// aqui não desfaz o jogo movido: só fica sem a imagem.
func moveSidecars(ctx context.Context, g library.Game, from, to string) {
	stem := g.Stem()
	for _, sub := range moveMedia {
		dir := filepath.Join(from, g.Folder, sub)
		items, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		for _, it := range items {
			name := it.Name()
			if it.IsDir() || strings.TrimSuffix(name, filepath.Ext(name)) != stem {
				continue
			}
			out := filepath.Join(to, g.Folder, sub, name)
			if _, err := os.Lstat(out); err == nil {
				continue
			}
			if err := moveEntry(ctx, filepath.Join(dir, name), out, func(int64) {}); err != nil {
				log.Printf("mover %s: %v", name, err)
			}
		}
	}
}

func humanBytes(n int64) string {
	const unit = 1024
	if n < unit {
		return fmt.Sprintf("%d B", n)
	}
	div, exp := int64(unit), 0
	for m := n / unit; m >= unit; m /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(n)/float64(div), "KMGTPE"[exp])
}
