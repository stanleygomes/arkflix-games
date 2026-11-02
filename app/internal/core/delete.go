package core

import (
	"bufio"
	"errors"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// GameSizeInfo é o que a confirmação de apagar mostra.
type GameSizeInfo struct {
	Size   int64 `json:"size"`   // quanto libera no disco
	Files  int   `json:"files"`  // arquivos/pastas que saem
	Server bool  `json:"server"` // está no RomM: volta como "no servidor"
}

// GameSize diz quanto apagar o jogo libera (com as trilhas e discos dele).
func (s *Service) GameSize(id string) (GameSizeInfo, error) {
	if err := s.adult(); err != nil {
		return GameSizeInfo{}, err
	}
	e, _, err := s.find(id)
	if err != nil {
		return GameSizeInfo{}, err
	}
	paths, size, err := s.deleteSet(e)
	if err != nil {
		return GameSizeInfo{}, err
	}
	return GameSizeInfo{Size: size, Files: len(paths), Server: e.romID > 0}, nil
}

// DeleteGame apaga o jogo do disco. Saves e pontos salvos ficam. Jogo que
// também está no servidor volta como "no servidor" na próxima varredura.
func (s *Service) DeleteGame(id string) error {
	if err := s.adult(); err != nil {
		return err
	}
	e, _, err := s.find(id)
	if err != nil {
		return err
	}
	if s.Running() == id {
		return errors.New("feche o jogo antes de apagar")
	}
	paths, _, err := s.deleteSet(e)
	if err != nil {
		return err
	}
	var failed error
	for _, p := range paths {
		if err := os.RemoveAll(p); err != nil && failed == nil {
			failed = err
		}
	}
	s.scan()
	s.opt.Emitter.Emit("library", s.views())
	return failed
}

// deleteSet: o que sai do disco ao apagar o jogo. Só dentro de uma pasta de
// biblioteca gravável, e só <raiz>/<console>/<jogo> (nunca a pasta do console).
// Pasta de jogo (vários discos) sai inteira; .cue e .m3u soltos levam as
// trilhas e discos que citam, se estiverem do lado.
func (s *Service) deleteSet(e entry) ([]string, int64, error) {
	g := e.game
	if e.remote || g.Entry == "" {
		return nil, 0, errors.New("este jogo não está neste PC")
	}
	root := ""
	for _, d := range s.opt.Store.Settings().LibraryDirs {
		if d.Writable && filepath.Clean(d.Path) == filepath.Clean(g.Root) {
			root = filepath.Clean(d.Path)
		}
	}
	if root == "" {
		return nil, 0, errors.New("este jogo está numa pasta só de leitura: apague por fora do app")
	}
	entry := filepath.Clean(g.Entry)
	rel, err := filepath.Rel(root, entry)
	if err != nil || strings.HasPrefix(rel, "..") || len(strings.Split(rel, string(filepath.Separator))) != 2 {
		return nil, 0, errors.New("o jogo está fora da pasta da biblioteca")
	}
	info, err := os.Lstat(entry)
	if err != nil {
		return nil, 0, errors.New("o arquivo do jogo não existe mais")
	}
	if info.IsDir() {
		return []string{entry}, treeSize(entry), nil
	}
	paths := []string{entry}
	seen := map[string]bool{entry: true}
	var add func(path string)
	add = func(path string) {
		for _, name := range referenced(path) {
			// só o nome: nada de seguir caminho pra fora da pasta do console
			p := filepath.Join(filepath.Dir(entry), filepath.Base(name))
			if seen[p] {
				continue
			}
			if st, err := os.Lstat(p); err != nil || !st.Mode().IsRegular() {
				continue
			}
			seen[p] = true
			paths = append(paths, p)
			add(p)
		}
	}
	add(entry)
	var size int64
	for _, p := range paths {
		if st, err := os.Lstat(p); err == nil {
			size += st.Size()
		}
	}
	return paths, size, nil
}

var cueRef = regexp.MustCompile(`(?i)^\s*FILE\s+"?([^"]+?)"?\s+\w+\s*$`)

// referenced: arquivos que um .cue (FILE) ou .m3u (uma linha por disco) cita.
func referenced(path string) []string {
	ext := strings.ToLower(filepath.Ext(path))
	if ext != ".cue" && ext != ".m3u" {
		return nil
	}
	f, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer f.Close()
	var out []string
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		switch {
		case ext == ".cue":
			if m := cueRef.FindStringSubmatch(line); m != nil {
				out = append(out, strings.ReplaceAll(m[1], `\`, "/"))
			}
		case line != "" && !strings.HasPrefix(line, "#"):
			out = append(out, strings.ReplaceAll(line, `\`, "/"))
		}
	}
	return out
}

func treeSize(path string) int64 {
	var total int64
	filepath.WalkDir(path, func(_ string, d os.DirEntry, err error) error {
		if err == nil && !d.IsDir() {
			if info, err := d.Info(); err == nil {
				total += info.Size()
			}
		}
		return nil
	})
	return total
}
