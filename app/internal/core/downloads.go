package core

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/stanleygomes/arkflix-games/app/internal/catalog"
	"github.com/stanleygomes/arkflix-games/app/internal/romm"
	"github.com/stanleygomes/arkflix-games/app/internal/settings"
)

// Fila de downloads do servidor: um jogo por vez, na ordem em que foram
// pedidos. Pausar guarda o .part e continuar segue de onde parou. Com "só de
// madrugada", a fila só anda entre 0h e 6h (ou num "baixar agora" do item).
// A fila fica em <cache>/romm/queue.json: ao abrir o app, o que sobrou volta
// pausado (nada começa sozinho).

// Estados de download além dos de toda tarefa (running, done, error, cancelled).
const (
	dlQueued  = "queued"  // na fila
	dlWaiting = "waiting" // esperando a madrugada
	dlPaused  = "paused"
	dlRunning = "running"
)

// dlTick: de quanto em quanto tempo a fila confere se a madrugada começou.
const dlTick = 30 * time.Second

type dlItem struct {
	ID       string  `json:"id"` // jogo na biblioteca
	RomID    int     `json:"romId"`
	Name     string  `json:"name"`
	System   string  `json:"system"`
	FsName   string  `json:"fsName"`
	Now      bool    `json:"now,omitempty"` // baixar mesmo fora da madrugada
	Progress float64 `json:"progress"`
	State    string  `json:"state"`

	cancel  context.CancelFunc
	pausing bool
}

type dlQueue struct {
	mu    sync.Mutex
	file  string
	items []*dlItem
	clock func() time.Time
}

func newDLQueue(file string, restore bool) *dlQueue {
	q := &dlQueue{file: file, clock: time.Now}
	if !restore {
		return q
	}
	var saved []*dlItem
	if raw, err := os.ReadFile(file); err == nil && json.Unmarshal(raw, &saved) == nil {
		for _, it := range saved {
			if it.ID != "" && it.RomID > 0 {
				it.State = dlPaused
				q.items = append(q.items, it)
			}
		}
	}
	return q
}

// isNight: entre 0h e 6h, na hora local.
func isNight(t time.Time) bool { return t.Hour() < 6 }

func (q *dlQueue) find(id string) (int, *dlItem) {
	for i, it := range q.items {
		if it.ID == id {
			return i, it
		}
	}
	return -1, nil
}

func (q *dlQueue) remove(it *dlItem) {
	q.items = slices.DeleteFunc(q.items, func(x *dlItem) bool { return x == it })
}

// save grava a fila (com o lock). Fila vazia apaga o arquivo.
func (q *dlQueue) save() {
	if len(q.items) == 0 {
		os.Remove(q.file)
		return
	}
	writePrivate(q.file, q.items)
}

func (it *dlItem) event() TaskEvent {
	ev := TaskEvent{ID: "download:" + it.ID, Label: "Baixando " + it.Name, State: it.State, Progress: it.Progress}
	switch it.State {
	case dlQueued:
		ev.Message = "na fila"
	case dlWaiting:
		ev.Message = "esperando a madrugada (0h às 6h)"
	case dlPaused:
		ev.Message = "pausado"
	}
	return ev
}

func (s *Service) emitAll(evs []TaskEvent) {
	for _, ev := range evs {
		s.opt.Emitter.Emit("task", ev)
	}
}

// dlLoop confere a fila de tempos em tempos (a madrugada começa sozinha).
func (s *Service) dlLoop() {
	t := time.NewTicker(dlTick)
	defer t.Stop()
	for range t.C {
		s.dlPump()
	}
}

// dlPump acerta o estado de quem espera e começa o próximo, se nada estiver
// baixando.
func (s *Service) dlPump() {
	cfg := s.opt.Store.Settings().Downloads
	q := s.dl
	q.mu.Lock()
	open := !cfg.NightOnly || isNight(q.clock())
	busy := slices.ContainsFunc(q.items, func(it *dlItem) bool { return it.State == dlRunning })
	var evs []TaskEvent
	var next *dlItem
	for _, it := range q.items {
		if it.State != "" && it.State != dlQueued && it.State != dlWaiting {
			continue
		}
		allowed := open || it.Now
		if allowed && !busy && next == nil {
			next = it
			continue
		}
		want := dlQueued
		if !allowed {
			want = dlWaiting
		}
		if it.State != want {
			it.State = want
			evs = append(evs, it.event())
		}
	}
	var ctx context.Context
	if next != nil {
		ctx, next.cancel = context.WithCancel(context.Background())
		next.State, next.pausing = dlRunning, false
		evs = append(evs, next.event())
	}
	if len(evs) > 0 {
		q.save()
	}
	q.mu.Unlock()
	s.emitAll(evs)
	if next != nil {
		go s.dlRun(ctx, next)
	}
}

// dlDir: pasta do console na pasta de importação (validada).
func (s *Service) dlDir(it *dlItem) (string, error) {
	dest := s.opt.Store.Settings().ImportDir
	if dest == "" {
		return "", errors.New("pasta de importação não configurada")
	}
	if _, ok := catalog.Find(it.System); !ok || !filepath.IsLocal(it.System) {
		return "", fmt.Errorf("console desconhecido: %q", it.System)
	}
	return filepath.Join(dest, it.System), nil
}

// dlDiscard apaga o download pela metade (só o .part do jogo, dentro da
// pasta de importação).
func (s *Service) dlDiscard(it *dlItem) {
	dir, err := s.dlDir(it)
	if err != nil {
		return
	}
	if part, err := romm.PartPath(dir, romm.Rom{FsName: it.FsName}); err == nil {
		os.RemoveAll(part)
	}
}

func (s *Service) dlRun(ctx context.Context, it *dlItem) {
	err := s.dlFetch(ctx, it)
	q := s.dl
	q.mu.Lock()
	it.cancel = nil
	var ev TaskEvent
	switch {
	case err == nil:
		q.remove(it)
		ev = TaskEvent{ID: "download:" + it.ID, Label: "Baixando " + it.Name, State: "done", Progress: 1, Message: it.Name + " baixado"}
	case ctx.Err() != nil && it.pausing:
		it.State, it.pausing = dlPaused, false
		ev = it.event()
	case ctx.Err() != nil:
		q.remove(it)
		s.dlDiscard(it)
		ev = TaskEvent{ID: "download:" + it.ID, Label: "Baixando " + it.Name, State: "cancelled", Message: "cancelado"}
	default:
		q.remove(it)
		s.dlDiscard(it)
		ev = TaskEvent{ID: "download:" + it.ID, Label: "Baixando " + it.Name, State: "error", Progress: it.Progress, Message: err.Error()}
	}
	q.save()
	q.mu.Unlock()
	if err == nil {
		s.scan()
		s.opt.Emitter.Emit("library", s.views())
	}
	s.opt.Emitter.Emit("task", ev)
	s.dlPump()
}

func (s *Service) dlFetch(ctx context.Context, it *dlItem) error {
	c, _ := s.remote.client()
	if c == nil {
		return errors.New("nenhum servidor ligado")
	}
	dir, err := s.dlDir(it)
	if err != nil {
		return err
	}
	c.Limit = func() int64 { return int64(s.opt.Store.Settings().Downloads.LimitKBps) * 1024 }
	rom, err := c.Rom(ctx, it.RomID)
	if err != nil {
		return err
	}
	var last time.Time
	_, err = c.Download(ctx, rom, dir, func(done, total int64) {
		if total <= 0 {
			return
		}
		s.dl.mu.Lock()
		it.Progress = min(float64(done)/float64(total), 1)
		ev := it.event()
		s.dl.mu.Unlock()
		if time.Since(last) < 200*time.Millisecond && ev.Progress < 1 {
			return
		}
		last = time.Now()
		s.opt.Emitter.Emit("task", ev)
	})
	return err
}

// DownloadGame põe um jogo do servidor na fila de downloads (pra pasta de
// importação).
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
	if c, _ := s.remote.client(); c == nil {
		return errors.New("nenhum servidor ligado")
	}
	dest := s.opt.Store.Settings().ImportDir
	if dest == "" {
		return errors.New("pasta de importação não configurada")
	}
	if free := freeSpace(dest); free >= 0 && free < e.game.Size+200<<20 {
		return fmt.Errorf("falta espaço em %s: o jogo tem %s", dest, humanSize(e.game.Size))
	}
	name := e.info.Name
	if name == "" {
		name = e.game.Title
	}
	q := s.dl
	q.mu.Lock()
	if _, it := q.find(id); it != nil {
		q.mu.Unlock()
		if it.State == dlPaused {
			return errors.New("este download está pausado: continue pela Central")
		}
		return errors.New("este jogo já está na fila de downloads")
	}
	q.items = append(q.items, &dlItem{ID: id, RomID: e.romID, Name: name, System: e.game.System, FsName: e.game.FileName})
	q.mu.Unlock()
	s.dlPump()
	return nil
}

// Downloads lista a fila (em andamento, na fila, esperando, pausados).
func (s *Service) Downloads() []TaskEvent {
	s.dl.mu.Lock()
	defer s.dl.mu.Unlock()
	out := []TaskEvent{}
	for _, it := range s.dl.items {
		if it.State != "" {
			out = append(out, it.event())
		}
	}
	return out
}

func (s *Service) dlGet(id string) (*dlItem, error) {
	_, it := s.dl.find(id)
	if it == nil {
		return nil, errors.New("esse download já terminou")
	}
	return it, nil
}

// PauseDownload pausa um download (o que já baixou fica guardado).
func (s *Service) PauseDownload(id string) error {
	if err := s.adult(); err != nil {
		return err
	}
	q := s.dl
	q.mu.Lock()
	it, err := s.dlGet(id)
	if err != nil {
		q.mu.Unlock()
		return err
	}
	switch it.State {
	case dlRunning:
		it.pausing = true
		cancel := it.cancel
		q.mu.Unlock()
		if cancel != nil {
			cancel()
		}
		return nil
	case dlPaused:
		q.mu.Unlock()
		return nil
	}
	it.State = dlPaused
	ev := it.event()
	q.save()
	q.mu.Unlock()
	s.opt.Emitter.Emit("task", ev)
	return nil
}

// ResumeDownload volta um download pausado pra fila (continua de onde parou).
func (s *Service) ResumeDownload(id string) error {
	if err := s.adult(); err != nil {
		return err
	}
	q := s.dl
	q.mu.Lock()
	it, err := s.dlGet(id)
	if err == nil && it.State == dlPaused {
		it.State = ""
	}
	q.mu.Unlock()
	if err != nil {
		return err
	}
	s.dlPump()
	return nil
}

// DownloadNow baixa este primeiro, mesmo fora da madrugada.
func (s *Service) DownloadNow(id string) error {
	if err := s.adult(); err != nil {
		return err
	}
	q := s.dl
	q.mu.Lock()
	i, it := q.find(id)
	if it == nil {
		q.mu.Unlock()
		return errors.New("esse download já terminou")
	}
	it.Now = true
	if it.State != dlRunning {
		if it.State == dlPaused {
			it.State = ""
		}
		q.items = append([]*dlItem{it}, slices.Delete(q.items, i, i+1)...)
	}
	q.mu.Unlock()
	s.dlPump()
	return nil
}

// dlCancel cancela um download da fila (handled = era download da fila).
func (s *Service) dlCancel(id string) (bool, error) {
	q := s.dl
	q.mu.Lock()
	_, it := q.find(id)
	if it == nil {
		q.mu.Unlock()
		return false, nil
	}
	if it.State == dlRunning {
		it.pausing = false
		cancel := it.cancel
		q.mu.Unlock()
		if cancel != nil {
			cancel()
		}
		return true, nil
	}
	q.remove(it)
	q.save()
	q.mu.Unlock()
	s.dlDiscard(it)
	s.opt.Emitter.Emit("task", TaskEvent{ID: "download:" + it.ID, Label: "Baixando " + it.Name, State: "cancelled", Message: "cancelado"})
	return true, nil
}

// dlClear esvazia a fila (servidor desligado).
func (s *Service) dlClear() {
	s.dl.mu.Lock()
	ids := make([]string, 0, len(s.dl.items))
	for _, it := range s.dl.items {
		ids = append(ids, it.ID)
	}
	s.dl.mu.Unlock()
	for _, id := range ids {
		s.dlCancel(id)
	}
}

// SetDownloadOptions grava as opções de download (velocidade e madrugada).
func (s *Service) SetDownloadOptions(d settings.Downloads) (settings.Settings, error) {
	if err := s.adult(); err != nil {
		return s.GetSettings(), err
	}
	next := s.opt.Store.Settings()
	next.Downloads = d
	if _, err := s.opt.Store.Save(next); err != nil {
		return s.GetSettings(), err
	}
	s.dlPump()
	return s.GetSettings(), nil
}

// downloadID: o id do jogo de uma tarefa "download:<id>".
func downloadID(task string) (string, bool) { return strings.CutPrefix(task, "download:") }
