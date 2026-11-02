package core

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/stanleygomes/arkflix-games/app/internal/catalog"
	"github.com/stanleygomes/arkflix-games/app/internal/library"
	"github.com/stanleygomes/arkflix-games/app/internal/meta"
	"github.com/stanleygomes/arkflix-games/app/internal/romm"
	"github.com/stanleygomes/arkflix-games/app/internal/saves"
	"github.com/stanleygomes/arkflix-games/app/internal/settings"
)

// remoteState é a ligação com um servidor RomM: a conta (token do app, nunca a
// senha), o catálogo em cache e as capas baixadas. Os jogos do servidor entram
// na biblioteca como "remotos" até serem baixados.
type remoteState struct {
	file string // <dados>/romm.json (0600)
	dir  string // <cache>/romm: catálogo e capas

	mu      sync.Mutex
	conn    *remoteConn
	roms    []romm.Rom
	updated int64
}

type remoteConn struct {
	romm.Account
	Owner     string `json:"owner"` // perfil cujos saves sincronizam
	Connected int64  `json:"connected"`
}

type remoteCatalog struct {
	Updated int64      `json:"updated"`
	Roms    []romm.Rom `json:"roms"`
}

// pastas que o RetroArch cria com "separar saves por core" (nome do core).
var raCoreDirs = map[string]string{
	"mesen": "Mesen", "snes9x": "Snes9x", "gambatte": "Gambatte", "genesis_plus_gx": "Genesis Plus GX",
	"mupen64plus_next": "Mupen64Plus-Next", "mednafen_psx": "Beetle PSX", "fbneo": "FinalBurn Neo",
	"melondsds": "melonDS DS",
}

func (r *remoteState) load() {
	r.mu.Lock()
	defer r.mu.Unlock()
	var c remoteConn
	if raw, err := os.ReadFile(r.file); err == nil && json.Unmarshal(raw, &c) == nil && c.Token != "" {
		r.conn = &c
	}
	var cat remoteCatalog
	if raw, err := os.ReadFile(filepath.Join(r.dir, "catalog.json")); err == nil && json.Unmarshal(raw, &cat) == nil && r.conn != nil {
		r.roms, r.updated = cat.Roms, cat.Updated
	}
}

func (r *remoteState) snapshot() (*remoteConn, []romm.Rom) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.conn, r.roms
}

func (r *remoteState) client() (*romm.Client, *remoteConn) {
	conn, _ := r.snapshot()
	if conn == nil {
		return nil, nil
	}
	return &romm.Client{Base: conn.Base, Token: conn.Token}, conn
}

func (r *remoteState) cover(id int) string {
	return filepath.Join(r.dir, "covers", fmt.Sprintf("%d.png", id))
}

func writePrivate(path string, v any) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	raw, _ := json.MarshalIndent(v, "", "  ")
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, raw, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

// remoteIndex: jogo do servidor por id da biblioteca (console + nome no disco).
func remoteIndex(roms []romm.Rom) map[string]romm.Rom {
	out := map[string]romm.Rom{}
	for _, r := range roms {
		if _, ok := catalog.Find(r.Platform); ok {
			out[library.ID(r.Platform, r.FsName)] = r
		}
	}
	return out
}

// remoteInfo: metadados do RomM pro jogo (a capa, se já baixada).
func (r *remoteState) info(rom romm.Rom) meta.Info {
	info := meta.Info{Name: rom.Name, Summary: rom.Summary, Year: rom.Year, Genres: rom.Genres, Players: rom.Players}
	if _, err := os.Stat(r.cover(rom.ID)); err == nil {
		info.Cover = r.cover(rom.ID)
	}
	return info
}

// mergeRemote liga os jogos locais aos do servidor e acrescenta os que só
// existem lá. O que falta no local (capa, descrição...) vem do RomM.
func (s *Service) mergeRemote(entries []entry) []entry {
	_, roms := s.remote.snapshot()
	if len(roms) == 0 {
		return entries
	}
	index := remoteIndex(roms)
	for i := range entries {
		rom, ok := index[entries[i].game.ID]
		if !ok {
			continue
		}
		delete(index, entries[i].game.ID)
		e := &entries[i]
		e.romID = rom.ID
		r := s.remote.info(rom)
		if e.auto.Cover == "" {
			e.auto.Cover = r.Cover
		}
		if e.auto.Summary == "" {
			e.auto.Summary, e.auto.Year, e.auto.Genres, e.auto.Players = r.Summary, r.Year, r.Genres, r.Players
		}
		e.info = e.edit.Apply(e.auto)
	}
	for id, rom := range index {
		g := library.Game{ID: id, System: rom.Platform, Title: library.CleanTitle(rom.FsName), FileName: rom.FsName, Folder: rom.Platform, Size: rom.Size}
		auto := s.remote.info(rom)
		edit := s.over.Get(id)
		entries = append(entries, entry{game: g, auto: auto, info: edit.Apply(auto), edit: edit, romID: rom.ID, remote: true})
	}
	return entries
}

// RemoteInfo é o estado da ligação com o servidor.
type RemoteInfo struct {
	Connected bool   `json:"connected"`
	Base      string `json:"base"`
	User      string `json:"user"`
	Owner     string `json:"owner"` // nome do perfil que sincroniza saves
	Mine      bool   `json:"mine"`  // o perfil em uso é o que sincroniza
	Games     int    `json:"games"`
	Updated   int64  `json:"updated"`
}

// RemoteStatus diz se há servidor ligado.
func (s *Service) RemoteStatus() RemoteInfo {
	conn, roms := s.remote.snapshot()
	if conn == nil {
		return RemoteInfo{}
	}
	s.remote.mu.Lock()
	updated := s.remote.updated
	s.remote.mu.Unlock()
	out := RemoteInfo{Connected: true, Base: conn.Base, User: conn.User, Mine: conn.Owner == s.cur().id, Updated: updated}
	if p, err := s.profiles.Get(conn.Owner); err == nil {
		out.Owner = p.Name
	}
	for _, r := range roms {
		if _, ok := catalog.Find(r.Platform); ok {
			out.Games++
		}
	}
	return out
}

// ConnectRemote entra no RomM com usuário e senha. A senha vira um token do
// app no servidor e é descartada; os saves do perfil em uso passam a sincronizar.
func (s *Service) ConnectRemote(host, user, pass string) (RemoteInfo, error) {
	if err := s.adult(); err != nil {
		return RemoteInfo{}, err
	}
	if strings.TrimSpace(user) == "" || pass == "" {
		return RemoteInfo{}, errors.New("informe usuário e senha")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	device, _ := os.Hostname()
	acc, err := romm.Login(ctx, host, strings.TrimSpace(user), pass, "arkflix-games ("+device+")")
	if err != nil {
		return RemoteInfo{}, err
	}
	conn := &remoteConn{Account: acc, Owner: s.cur().id, Connected: time.Now().Unix()}
	if err := writePrivate(s.remote.file, conn); err != nil {
		return RemoteInfo{}, err
	}
	s.remote.mu.Lock()
	s.remote.conn, s.remote.roms = conn, nil
	s.remote.mu.Unlock()
	s.startRemoteRefresh()
	return s.RemoteStatus(), nil
}

// DisconnectRemote esquece o servidor (o token do app continua no RomM até
// ser apagado lá) e tira os jogos remotos da biblioteca.
func (s *Service) DisconnectRemote() error {
	if err := s.adult(); err != nil {
		return err
	}
	if err := os.Remove(s.remote.file); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	os.RemoveAll(s.remote.dir)
	s.remote.mu.Lock()
	s.remote.conn, s.remote.roms, s.remote.updated = nil, nil, 0
	s.remote.mu.Unlock()
	s.scan()
	s.opt.Emitter.Emit("library", s.views())
	return nil
}

// RefreshRemote busca o catálogo do servidor de novo (em segundo plano).
func (s *Service) RefreshRemote() error {
	if conn, _ := s.remote.snapshot(); conn == nil {
		return errors.New("nenhum servidor ligado")
	}
	if !s.startRemoteRefresh() {
		return errors.New("a lista do servidor já está sendo atualizada")
	}
	return nil
}

func (s *Service) startRemoteRefresh() bool {
	return s.runTask("remote", "Jogos do servidor", func(progress func(float64)) (string, error) {
		c, _ := s.remote.client()
		if c == nil {
			return "", errors.New("nenhum servidor ligado")
		}
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()
		roms, err := c.Roms(ctx)
		if err != nil {
			return "", err
		}
		now := time.Now().Unix()
		if err := writePrivate(filepath.Join(s.remote.dir, "catalog.json"), remoteCatalog{Updated: now, Roms: roms}); err != nil {
			return "", err
		}
		s.remote.mu.Lock()
		s.remote.roms, s.remote.updated = roms, now
		s.remote.mu.Unlock()
		for i, r := range roms {
			if dest := s.remote.cover(r.ID); r.Cover != "" {
				if _, err := os.Stat(dest); err != nil {
					if err := c.Cover(ctx, r.Cover, dest); err != nil {
						log.Printf("capa do servidor (%s): %v", r.Name, err)
					}
				}
			}
			progress(float64(i+1) / float64(len(roms)))
		}
		s.scan()
		s.opt.Emitter.Emit("library", s.views())
		return fmt.Sprintf("%d jogos no servidor", s.RemoteStatus().Games), nil
	})
}

// DownloadGame baixa um jogo do servidor pra pasta de importação.
func (s *Service) DownloadGame(id string) error {
	if err := s.adult(); err != nil {
		return err
	}
	e, _, err := s.find(id)
	if err != nil {
		return err
	}
	if !e.remote {
		return errors.New("este jogo já está neste PC")
	}
	c, _ := s.remote.client()
	if c == nil {
		return errors.New("nenhum servidor ligado")
	}
	dest := s.opt.Store.Settings().ImportDir
	if dest == "" {
		return errors.New("pasta de importação não configurada")
	}
	dir := filepath.Join(dest, e.game.System)
	if free := freeSpace(dest); free >= 0 && free < e.game.Size+200<<20 {
		return fmt.Errorf("falta espaço em %s: o jogo tem %s", dest, humanSize(e.game.Size))
	}
	name := e.info.Name
	if name == "" {
		name = e.game.Title
	}
	ok := s.runTaskCtx("download:"+id, "Baixando "+name, func(ctx context.Context, progress func(float64)) (string, error) {
		rom, err := c.Rom(ctx, e.romID)
		if err != nil {
			return "", err
		}
		_, err = c.Download(ctx, rom, dir, func(done, total int64) {
			if total > 0 {
				progress(float64(done) / float64(total))
			}
		})
		if err != nil {
			return "", err
		}
		s.scan()
		s.opt.Emitter.Emit("library", s.views())
		return name + " baixado", nil
	})
	if !ok {
		return errors.New("este jogo já está sendo baixado")
	}
	return nil
}

func humanSize(n int64) string {
	switch {
	case n >= 1<<30:
		return fmt.Sprintf("%.1f GB", float64(n)/(1<<30))
	case n >= 1<<20:
		return fmt.Sprintf("%.0f MB", float64(n)/(1<<20))
	}
	return fmt.Sprintf("%d KB", n/1024)
}

// syncs: o jogo sincroniza saves? Só RetroArch (PSP e PS2 guardam em pastas e
// memory cards, fora do formato do RomM), só jogo que também está no servidor
// e só no perfil que ligou o servidor.
func (s *Service) syncs(e entry, sys catalog.System, u *user) bool {
	conn, _ := s.remote.snapshot()
	return conn != nil && conn.Owner == u.id && e.romID > 0 && !e.remote && sys.Emulator == catalog.RetroArch
}

var syncSave = regexp.MustCompile(`^(.*)\.(srm|rtc|sav)$`)
var syncState = regexp.MustCompile(`^(.*)\.state(\d*|\.auto)$`)

func stemOf(path string) string {
	base := filepath.Base(path)
	return strings.TrimSuffix(base, filepath.Ext(base))
}

// localTarget: onde o save do servidor entra (o arquivo local, se existir; se
// não, a pasta do core, como o RetroArch do Flatpak separa).
func localTarget(dir, core, name string) (string, int64) {
	candidates := []string{filepath.Join(dir, name)}
	if entries, err := os.ReadDir(dir); err == nil {
		for _, d := range entries {
			if d.IsDir() {
				candidates = append(candidates, filepath.Join(dir, d.Name(), name))
			}
		}
	}
	for _, p := range candidates {
		if info, err := os.Stat(p); err == nil {
			return p, info.ModTime().Unix()
		}
	}
	if sub := raCoreDirs[core]; sub != "" {
		return filepath.Join(dir, sub, name), -1
	}
	return filepath.Join(dir, name), -1
}

// pullSaves baixa os saves/states do servidor que estejam mais novos que os
// daqui (ou que faltem). Nunca troca um arquivo daqui por um mais velho.
// Conflito (os dois mudaram desde a última sincronização, last): fica o
// daqui e o do servidor vai pro lado, com ".servidor" no fim.
func (s *Service) pullSaves(ctx context.Context, e entry, sys catalog.System, dirs settings.SaveDirs, last int64) (int, []string, error) {
	c, _ := s.remote.client()
	if c == nil {
		return 0, nil, nil
	}
	var conflicts []string
	stem := stemOf(e.game.Launch)
	n := 0
	for _, kind := range []romm.Kind{romm.Saves, romm.States} {
		list, err := c.Assets(ctx, kind, e.romID)
		if err != nil {
			return n, conflicts, err
		}
		dir, re := dirs.RetroArchSaves, syncSave
		if kind == romm.States {
			dir, re = dirs.RetroArchStates, syncState
		}
		for _, a := range list {
			m := re.FindStringSubmatch(a.FileName)
			if m == nil || m[1] != stem || strings.ContainsAny(a.FileName, `/\`) {
				continue // de outro jogo, ou do player web do RomM (nome com data)
			}
			target, local := localTarget(dir, sys.Core, a.FileName)
			if local >= a.Updated.Unix()-1 {
				continue
			}
			if conflict(local, a.Updated.Unix(), last) {
				if err := c.Fetch(ctx, kind, a, target+".servidor"); err != nil {
					return n, conflicts, err
				}
				conflicts = append(conflicts, a.FileName)
				continue
			}
			if err := c.Fetch(ctx, kind, a, target); err != nil {
				return n, conflicts, err
			}
			n++
		}
	}
	return n, conflicts, nil
}

// conflict: o arquivo daqui e o do servidor mudaram depois da última
// sincronização (jogou nos dois aparelhos sem sincronizar no meio).
func conflict(local, remote, last int64) bool {
	return last > 0 && local > last+2 && remote > last+2
}

// pushSaves envia o que mudou desde since (saves do jogo e states, com a
// miniatura). O servidor substitui pelo nome.
func (s *Service) pushSaves(ctx context.Context, e entry, sys catalog.System, l saves.Listing, since int64) (int, error) {
	c, _ := s.remote.client()
	if c == nil {
		return 0, nil
	}
	n := 0
	for _, f := range l.Files {
		if f.Shared || f.Modified < since {
			continue
		}
		if err := c.Upload(ctx, romm.Saves, e.romID, sys.Core, f.Path, ""); err != nil {
			return n, err
		}
		n++
	}
	for _, st := range l.States {
		if st.Modified < since {
			continue
		}
		shot := ""
		if st.Thumb {
			shot = st.Path + ".png"
		}
		if err := c.Upload(ctx, romm.States, e.romID, sys.Core, st.Path, shot); err != nil {
			return n, err
		}
		n++
	}
	return n, nil
}
