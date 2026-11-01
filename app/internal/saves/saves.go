// Package saves acha e apaga os saves de cada jogo: save states (com
// miniatura) e saves de bateria/memory card. Implementa core.SaveStore.
package saves

import (
	"archive/zip"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/stanleygomes/arkflix-games/app/internal/catalog"
	"github.com/stanleygomes/arkflix-games/app/internal/isoinfo"
	"github.com/stanleygomes/arkflix-games/app/internal/settings"
)

// State é um save state.
type State struct {
	Path     string `json:"-"`
	Token    string `json:"id"`   // identifica o arquivo pra interface
	Slot     int    `json:"slot"` // -1 = automático (ao sair do jogo)
	Label    string `json:"label"`
	Modified int64  `json:"modified"`
	Thumb    bool   `json:"thumb"`
}

// File é um save de bateria, pasta SAVEDATA ou memory card.
type File struct {
	Path     string `json:"path"`
	Name     string `json:"name"`
	Size     int64  `json:"size"`
	Modified int64  `json:"modified"`
	Shared   bool   `json:"shared"` // memory card usado por vários jogos
}

// Game é o que o pacote precisa saber do jogo.
type Game struct {
	System catalog.System
	Launch string // arquivo que o emulador abre
}

// Listing são os saves de um jogo.
type Listing struct {
	States []State `json:"states"`
	Files  []File  `json:"files"`
	Note   string  `json:"note,omitempty"`
}

func stem(path string) string {
	base := filepath.Base(path)
	return strings.TrimSuffix(base, filepath.Ext(base))
}

func modTime(info os.FileInfo) int64 { return info.ModTime().Unix() }

// List devolve os saves do jogo, states mais recentes primeiro.
func List(g Game, dirs settings.SaveDirs) Listing {
	var l Listing
	switch g.System.Emulator {
	case catalog.RetroArch:
		l = retroarch(g, dirs)
	case catalog.PPSSPP:
		l = ppsspp(g, dirs)
	case catalog.PCSX2:
		l = pcsx2(g, dirs)
	}
	sort.SliceStable(l.States, func(i, j int) bool { return l.States[i].Modified > l.States[j].Modified })
	return l
}

var raState = regexp.MustCompile(`\.state(\d*|\.auto)$`)

func retroarch(g Game, dirs settings.SaveDirs) Listing {
	var l Listing
	prefix := stem(g.Launch)
	entries, _ := os.ReadDir(dirs.RetroArchStates)
	for _, e := range entries {
		name := e.Name()
		rest, ok := strings.CutPrefix(name, prefix)
		if !ok || e.IsDir() {
			continue
		}
		m := raState.FindStringSubmatch(rest)
		if m == nil || m[0] != rest {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		st := State{Path: filepath.Join(dirs.RetroArchStates, name), Modified: modTime(info)}
		switch m[1] {
		case ".auto":
			st.Slot, st.Label = -1, "Automático (ao sair)"
		case "":
			st.Slot, st.Label = 0, "Slot 0"
		default:
			st.Slot, _ = strconv.Atoi(m[1])
			st.Label = fmt.Sprintf("Slot %d", st.Slot)
		}
		_, err = os.Stat(st.Path + ".png")
		st.Thumb = err == nil
		l.States = append(l.States, st)
	}
	for _, ext := range []string{".srm", ".rtc", ".sav"} {
		path := filepath.Join(dirs.RetroArchSaves, prefix+ext)
		if info, err := os.Stat(path); err == nil {
			l.Files = append(l.Files, File{Path: path, Name: filepath.Base(path), Size: info.Size(), Modified: modTime(info)})
		}
	}
	return l
}

var pspState = regexp.MustCompile(`^(.+)_(\d+\.\d+)_(\d+)\.ppst$`)

func ppsspp(g Game, dirs settings.SaveDirs) Listing {
	var l Listing
	if !strings.EqualFold(filepath.Ext(g.Launch), ".iso") {
		l.Note = "Saves de PSP só aparecem pra jogos em .iso"
		return l
	}
	info, err := isoinfo.ReadFile(g.Launch)
	if err != nil || info.Serial == "" {
		l.Note = "Não deu pra ler o código do jogo na ISO"
		return l
	}
	entries, _ := os.ReadDir(dirs.PPSSPPStates)
	for _, e := range entries {
		m := pspState.FindStringSubmatch(e.Name())
		if m == nil || m[1] != info.Serial {
			continue
		}
		fi, err := e.Info()
		if err != nil {
			continue
		}
		slot, _ := strconv.Atoi(m[3])
		path := filepath.Join(dirs.PPSSPPStates, e.Name())
		_, err = os.Stat(strings.TrimSuffix(path, ".ppst") + ".jpg")
		l.States = append(l.States, State{Path: path, Slot: slot, Label: fmt.Sprintf("Slot %d", slot+1), Modified: modTime(fi), Thumb: err == nil})
	}
	saveDirs, _ := os.ReadDir(dirs.PPSSPPSaveData)
	for _, e := range saveDirs {
		if !e.IsDir() || !strings.HasPrefix(e.Name(), info.Serial) {
			continue
		}
		path := filepath.Join(dirs.PPSSPPSaveData, e.Name())
		fi, _ := e.Info()
		l.Files = append(l.Files, File{Path: path, Name: e.Name(), Size: dirSize(path), Modified: modTime(fi)})
	}
	return l
}

var ps2State = regexp.MustCompile(`^(\S+) \(([0-9A-F]{8})\)\.(\d{2}|resume)\.p2s$`)

func pcsx2(g Game, dirs settings.SaveDirs) Listing {
	var l Listing
	cards, _ := filepath.Glob(filepath.Join(dirs.PCSX2Memcards, "*.ps2"))
	for _, c := range cards {
		if info, err := os.Stat(c); err == nil {
			l.Files = append(l.Files, File{Path: c, Name: filepath.Base(c), Size: info.Size(), Modified: modTime(info), Shared: true})
		}
	}
	if !strings.EqualFold(filepath.Ext(g.Launch), ".iso") {
		l.Note = "Save states de PS2 só aparecem pra jogos em .iso"
		return l
	}
	info, err := isoinfo.ReadFile(g.Launch)
	if err != nil || info.Serial == "" {
		l.Note = "Não deu pra ler o código do jogo na ISO"
		return l
	}
	entries, _ := os.ReadDir(dirs.PCSX2States)
	for _, e := range entries {
		m := ps2State.FindStringSubmatch(e.Name())
		if m == nil || m[1] != info.Serial {
			continue
		}
		fi, err := e.Info()
		if err != nil {
			continue
		}
		st := State{Path: filepath.Join(dirs.PCSX2States, e.Name()), Modified: modTime(fi)}
		if m[3] == "resume" {
			st.Slot, st.Label = -1, "Automático (ao sair)"
		} else {
			st.Slot, _ = strconv.Atoi(m[3])
			st.Label = fmt.Sprintf("Slot %d", st.Slot)
		}
		st.Thumb = p2sHasScreenshot(st.Path)
		l.States = append(l.States, st)
	}
	return l
}

func p2sHasScreenshot(path string) bool {
	z, err := zip.OpenReader(path)
	if err != nil {
		return false
	}
	defer z.Close()
	for _, f := range z.File {
		if f.Name == "Screenshot.png" {
			return true
		}
	}
	return false
}

func dirSize(path string) int64 {
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

// Thumbnail abre a miniatura do state (png do RetroArch, jpg do PPSSPP,
// Screenshot.png dentro do .p2s do PCSX2). Devolve o conteúdo e o tipo.
func Thumbnail(statePath string) ([]byte, string, error) {
	switch {
	case strings.HasSuffix(statePath, ".p2s"):
		z, err := zip.OpenReader(statePath)
		if err != nil {
			return nil, "", err
		}
		defer z.Close()
		for _, f := range z.File {
			if f.Name == "Screenshot.png" {
				rc, err := f.Open()
				if err != nil {
					return nil, "", err
				}
				defer rc.Close()
				data, err := io.ReadAll(io.LimitReader(rc, 8<<20))
				return data, "image/png", err
			}
		}
		return nil, "", os.ErrNotExist
	case strings.HasSuffix(statePath, ".ppst"):
		data, err := os.ReadFile(strings.TrimSuffix(statePath, ".ppst") + ".jpg")
		return data, "image/jpeg", err
	default:
		data, err := os.ReadFile(statePath + ".png")
		return data, "image/png", err
	}
}

// Inside diz se path está dentro de alguma pasta de save (proteção ao apagar
// ou servir arquivo pedido pela interface).
func Inside(path string, dirs settings.SaveDirs) bool {
	clean := filepath.Clean(path)
	for _, d := range []string{dirs.RetroArchStates, dirs.RetroArchSaves, dirs.PPSSPPStates, dirs.PPSSPPSaveData, dirs.PCSX2States, dirs.PCSX2Memcards} {
		if d == "" {
			continue
		}
		rel, err := filepath.Rel(filepath.Clean(d), clean)
		if err == nil && rel != "." && !strings.HasPrefix(rel, "..") {
			return true
		}
	}
	return false
}

// Delete apaga o state e a miniatura dele.
func Delete(statePath string, dirs settings.SaveDirs) error {
	if !Inside(statePath, dirs) {
		return errors.New("arquivo fora das pastas de save")
	}
	if err := os.Remove(statePath); err != nil {
		return err
	}
	for _, thumb := range []string{statePath + ".png", strings.TrimSuffix(statePath, ".ppst") + ".jpg"} {
		if thumb != statePath {
			os.Remove(thumb)
		}
	}
	return nil
}

// Local guarda os saves nas pastas dos emuladores neste PC. Um servidor no
// futuro implementa a mesma interface (core.SaveStore).
type Local struct {
	Dirs func() settings.SaveDirs
}

// List lista os saves do jogo.
func (l Local) List(g Game) Listing { return List(g, l.Dirs()) }

// Delete apaga um save state.
func (l Local) Delete(path string) error { return Delete(path, l.Dirs()) }

// Owns diz se o caminho é um save deste armazenamento.
func (l Local) Owns(path string) bool { return Inside(path, l.Dirs()) }

// Thumbnail lê a miniatura de um save state.
func (l Local) Thumbnail(path string) ([]byte, string, error) {
	if !l.Owns(path) {
		return nil, "", errors.New("arquivo fora das pastas de save")
	}
	return Thumbnail(path)
}
