package core

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"time"

	"github.com/stanleygomes/arkflix-games/app/internal/updater"
)

// UpdateOptions liga a atualização automática.
type UpdateOptions struct {
	// Check: olha a release nova uns segundos depois de abrir (desktop).
	Check bool
	// API: base da API do GitHub ("" = api.github.com; testes: servidor falso).
	API string
	// DryRun: baixa e confere, mas não instala ao fechar (ambiente falso).
	DryRun bool
	// Current: versão comparada com a release ("" = Options.Version). O
	// ambiente falso roda como "dev", que nunca atualiza.
	Current string
}

// UpdateInfo é o estado da atualização que a interface mostra.
type UpdateInfo struct {
	Current   string  `json:"current"`
	Latest    string  `json:"latest"`
	Available bool    `json:"available"`
	Notes     string  `json:"notes"` // texto da release (Markdown)
	URL       string  `json:"url"`   // página da release
	Asset     string  `json:"asset"` // instalador deste sistema ("" = não tem)
	Size      int64   `json:"size"`
	State     string  `json:"state"` // idle, downloading, ready, error
	Progress  float64 `json:"progress"`
	Error     string  `json:"error,omitempty"`
}

// checkDelay: espera antes da checagem do início (não disputa com a abertura).
var checkDelay = 5 * time.Second

// readyFile marca o download conferido: sobrevive a reabrir o app sem rede.
const readyFile = "ready.json"

type readyMark struct {
	Version string `json:"version"`
	File    string `json:"file"`
}

type updateState struct {
	mu        sync.Mutex
	info      UpdateInfo
	asset     updater.Asset
	sum       updater.Asset
	file      string // baixado e conferido (state ready)
	dir       string
	client    updater.Client
	installer func() (updater.Installer, error)
	goos      string
	goarch    string
}

func (s *Service) newUpdateState() *updateState {
	current := s.opt.Update.Current
	if current == "" {
		current = s.opt.Version
	}
	u := &updateState{
		info:      UpdateInfo{Current: current, State: "idle"},
		dir:       filepath.Join(s.opt.Cache, "update"),
		client:    updater.Client{API: s.opt.Update.API},
		installer: func() (updater.Installer, error) { return updater.Current(runtime.GOOS) },
		goos:      runtime.GOOS,
		goarch:    runtime.GOARCH,
	}
	u.loadReady()
	return u
}

// loadReady retoma o download pronto de outra execução; se o app já está
// nessa versão (instalou), limpa a pasta.
func (u *updateState) loadReady() {
	var m readyMark
	data, err := os.ReadFile(filepath.Join(u.dir, readyFile))
	if err == nil && json.Unmarshal(data, &m) == nil && updater.Newer(m.Version, u.info.Current) {
		file := filepath.Join(u.dir, filepath.Base(m.File))
		if _, err := os.Stat(file); err == nil {
			u.file = file
			u.info.Latest, u.info.Available, u.info.State, u.info.Progress = m.Version, true, "ready", 1
			u.info.Asset = filepath.Base(file)
			return
		}
	}
	os.RemoveAll(u.dir)
}

func (u *updateState) newInstaller() (updater.Installer, error) {
	u.mu.Lock()
	fn := u.installer
	u.mu.Unlock()
	return fn()
}

func (u *updateState) snapshot() UpdateInfo {
	u.mu.Lock()
	defer u.mu.Unlock()
	return u.info
}

// set muda o estado e devolve a cópia pra emitir.
func (u *updateState) set(fn func(*UpdateInfo)) UpdateInfo {
	u.mu.Lock()
	defer u.mu.Unlock()
	fn(&u.info)
	return u.info
}

// UpdateStatus devolve o estado da atualização (sem rede).
func (s *Service) UpdateStatus() UpdateInfo { return s.upd.snapshot() }

// CheckUpdate pergunta ao GitHub qual é a release mais nova.
func (s *Service) CheckUpdate() (UpdateInfo, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	rel, err := s.upd.client.Latest(ctx)
	if err != nil {
		return s.UpdateStatus(), fmt.Errorf("não deu pra ver se tem versão nova: %w", err)
	}
	info := s.upd.applyRelease(rel)
	if info.Available {
		s.opt.Emitter.Emit("update", info)
	}
	return info, nil
}

// applyRelease anota a release; download pronto de outra versão é descartado.
func (u *updateState) applyRelease(rel updater.Release) UpdateInfo {
	u.mu.Lock()
	defer u.mu.Unlock()
	name := updater.AssetName(u.goos, u.goarch)
	asset, okA := rel.Find(name)
	sum, okS := rel.Find(name + ".sha256")
	i := &u.info
	i.Latest, i.Notes, i.URL = rel.Tag, rel.Body, rel.HTMLURL
	i.Available = updater.Newer(rel.Tag, i.Current)
	i.Asset, i.Size, u.asset, u.sum = "", 0, updater.Asset{}, updater.Asset{}
	if okA && okS {
		i.Asset, i.Size, u.asset, u.sum = asset.Name, asset.Size, asset, sum
	}
	if i.State == "downloading" {
		return *i
	}
	if i.State == "ready" && i.Available && filepath.Base(u.file) == i.Asset && u.readyVersion() == rel.Tag {
		return *i
	}
	i.State, i.Progress, i.Error, u.file = "idle", 0, "", ""
	return *i
}

func (u *updateState) readyVersion() string {
	var m readyMark
	data, _ := os.ReadFile(filepath.Join(u.dir, readyFile))
	json.Unmarshal(data, &m)
	return m.Version
}

// DownloadUpdate baixa e confere a versão nova em segundo plano (tarefa
// "update"); instala quando o app fechar.
func (s *Service) DownloadUpdate() error {
	if err := s.adult(); err != nil {
		return err
	}
	return s.startDownload()
}

func (s *Service) startDownload() error {
	info := s.UpdateStatus()
	switch {
	case !info.Available:
		return errors.New("o app já está na versão mais nova")
	case info.State == "ready":
		return nil
	case info.Asset == "":
		return errors.New("a release não tem instalador pra este sistema")
	}
	if err := s.checkInstall(); err != nil {
		s.opt.Emitter.Emit("update", s.upd.set(func(i *UpdateInfo) { i.State, i.Error = "error", err.Error() }))
		return err
	}
	if !s.runTask("update", "Baixando atualização", s.download) {
		return errors.New("a atualização já está sendo baixada")
	}
	return nil
}

// checkInstall: no Linux sem permissão na pasta do app nem adianta baixar.
func (s *Service) checkInstall() error {
	if s.opt.Update.DryRun {
		return nil
	}
	in, err := s.upd.newInstaller()
	if err != nil {
		return err
	}
	return in.Check()
}

func (s *Service) download(progress func(float64)) (string, error) {
	u := s.upd
	u.mu.Lock()
	asset, sum, tag := u.asset, u.sum, u.info.Latest
	u.info.State, u.info.Progress, u.info.Error = "downloading", 0, ""
	u.mu.Unlock()
	s.opt.Emitter.Emit("update", u.snapshot())

	file, err := u.fetch(asset, sum, tag, func(p float64) {
		u.set(func(i *UpdateInfo) { i.Progress = p })
		progress(p)
	})
	if err != nil {
		s.opt.Emitter.Emit("update", u.set(func(i *UpdateInfo) { i.State, i.Error = "error", err.Error() }))
		return "", err
	}
	u.mu.Lock()
	u.file = file
	u.info.State, u.info.Progress = "ready", 1
	u.mu.Unlock()
	s.opt.Emitter.Emit("update", u.snapshot())
	return fmt.Sprintf("versão %s pronta: instala quando o app fechar", tag), nil
}

// fetch baixa o anexo conferindo o sha256 e marca como pronto.
func (u *updateState) fetch(asset, sum updater.Asset, tag string, progress func(float64)) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()
	want, err := u.client.Sum(ctx, sum.URL)
	if err != nil {
		return "", fmt.Errorf("checksum da atualização: %w", err)
	}
	os.RemoveAll(u.dir) // sobra de versão anterior
	file := filepath.Join(u.dir, asset.Name)
	if err := u.client.Download(ctx, asset.URL, want, file, asset.Size, progress); err != nil {
		return "", fmt.Errorf("download da atualização: %w", err)
	}
	mark, _ := json.Marshal(readyMark{Version: tag, File: asset.Name})
	if err := os.WriteFile(filepath.Join(u.dir, readyFile), mark, 0o600); err != nil {
		return "", err
	}
	return file, nil
}

// checkLater: checagem única do início; com "Atualizar sozinho" ligado,
// já baixa.
func (s *Service) checkLater() {
	time.Sleep(checkDelay)
	info, err := s.CheckUpdate()
	if err != nil {
		log.Printf("atualização: %v", err)
		return
	}
	if info.Available && info.State == "idle" && s.opt.Store.Settings().AutoUpdate {
		if err := s.startDownload(); err != nil {
			log.Printf("atualização: %v", err)
		}
	}
}

// ApplyUpdateOnQuit instala a atualização baixada; o desktop chama ao fechar.
// É função, não método, de propósito: método de Service vira API da
// interface (Wails e HTTP) e instalar não pode ser chamado de lá.
func ApplyUpdateOnQuit(s *Service) {
	u := s.upd
	u.mu.Lock()
	ready, file, tag := u.info.State == "ready", u.file, u.info.Latest
	u.mu.Unlock()
	if !ready {
		return
	}
	if s.opt.Update.DryRun {
		log.Printf("atualização %s pronta (ambiente falso: não instala)", tag)
		return
	}
	err := s.install(file)
	if err != nil {
		log.Printf("atualização %s não instalada: %v", tag, err)
		u.set(func(i *UpdateInfo) { i.State, i.Error = "error", err.Error() })
		return
	}
	log.Printf("atualização %s instalada", tag)
}

func (s *Service) install(file string) error {
	in, err := s.upd.newInstaller()
	if err != nil {
		return err
	}
	if err := in.Apply(file); err != nil {
		return err
	}
	// no Windows o setup ainda vai rodar: a pasta fica e a próxima
	// abertura limpa (a versão já não é mais nova)
	if in.GOOS != "windows" {
		os.RemoveAll(s.upd.dir)
	}
	return nil
}
