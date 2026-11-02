package core_test

import (
	"bytes"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stanleygomes/arkflix-games/app/internal/bt"
	"github.com/stanleygomes/arkflix-games/app/internal/core"
	"github.com/stanleygomes/arkflix-games/app/internal/fixture"
	"github.com/stanleygomes/arkflix-games/app/internal/hooks"
	"github.com/stanleygomes/arkflix-games/app/internal/launcher"
	"github.com/stanleygomes/arkflix-games/app/internal/settings"
)

// connect liga o RomM falso e espera o catálogo.
func connect(t *testing.T, h harness, fake *fixture.FakeRomM) {
	t.Helper()
	srv := httptest.NewServer(fake)
	t.Cleanup(srv.Close)
	if _, err := h.svc.ConnectRemote(srv.URL, "stanley", "senha123"); err != nil {
		t.Fatal(err)
	}
	if ev := h.rec.waitTask(t, "remote"); ev.State != "done" {
		t.Fatalf("catálogo: %+v", ev)
	}
}

// waitEvent espera um evento "task" que satisfaça ok; devolve a posição dele.
func (r *recorder) waitTaskEvent(t *testing.T, what string, ok func(core.TaskEvent) bool) int {
	t.Helper()
	deadline := time.Now().Add(15 * time.Second)
	for time.Now().Before(deadline) {
		r.mu.Lock()
		for i, e := range r.events {
			if ev, is := r.data[i].(core.TaskEvent); is && e == "task" && ok(ev) {
				r.mu.Unlock()
				return i
			}
		}
		r.mu.Unlock()
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("evento não chegou: %s", what)
	return -1
}

func state(id, s string) func(core.TaskEvent) bool {
	return func(ev core.TaskEvent) bool { return ev.ID == "download:"+id && ev.State == s }
}

func TestDownloadQueue(t *testing.T) {
	h := setup(t)
	fake := fixture.NewFakeRomM()
	fake.Slow = 20 * time.Millisecond
	connect(t, h, fake)
	dkc, spyro := h.game(t, "Donkey Kong Country"), h.game(t, "Spyro the Dragon")

	if err := h.svc.DownloadGame(dkc.ID); err != nil {
		t.Fatal(err)
	}
	if err := h.svc.DownloadGame(spyro.ID); err != nil {
		t.Fatal(err)
	}
	if err := h.svc.DownloadGame(spyro.ID); err == nil || !strings.Contains(err.Error(), "fila") {
		t.Fatalf("o mesmo jogo duas vezes: %v", err)
	}
	h.rec.waitTaskEvent(t, "Spyro na fila", state(spyro.ID, "queued"))
	if list := h.svc.Downloads(); len(list) != 2 || list[0].State != "running" || list[1].State != "queued" {
		t.Fatalf("fila: %+v", list)
	}
	dkcDone := h.rec.waitTaskEvent(t, "DKC baixado", state(dkc.ID, "done"))
	spyroRun := h.rec.waitTaskEvent(t, "Spyro baixando", state(spyro.ID, "running"))
	if spyroRun < dkcDone {
		t.Fatal("um por vez, na ordem: Spyro só começa depois do DKC")
	}
	if ev := h.rec.waitTask(t, "download:"+spyro.ID); ev.State != "done" {
		t.Fatalf("Spyro: %+v", ev)
	}
	if g := h.game(t, "Spyro the Dragon"); g.Remote {
		t.Fatal("Spyro baixado")
	}
	if len(h.svc.Downloads()) != 0 {
		t.Fatal("fila vazia no fim")
	}
}

func TestDownloadPauseResume(t *testing.T) {
	h := setup(t)
	fake := fixture.NewFakeRomM()
	big := make([]byte, 128<<10)
	for i := range big {
		big[i] = byte(i*31 + i/251)
	}
	fake.SetRom(1, big)
	connect(t, h, fake)
	dkc := h.game(t, "Donkey Kong Country")
	if _, err := h.svc.SetDownloadOptions(settings.Downloads{LimitKBps: 32}); err != nil {
		t.Fatal(err)
	}

	if err := h.svc.DownloadGame(dkc.ID); err != nil {
		t.Fatal(err)
	}
	h.rec.waitTaskEvent(t, "metade do caminho", func(ev core.TaskEvent) bool {
		return ev.ID == "download:"+dkc.ID && ev.State == "running" && ev.Progress >= 0.15
	})
	if err := h.svc.PauseDownload(dkc.ID); err != nil {
		t.Fatal(err)
	}
	h.rec.waitTaskEvent(t, "pausado", state(dkc.ID, "paused"))
	part := filepath.Join(h.env.Settings.ImportDir, "snes", "Donkey Kong Country (U).zip.part")
	st, err := os.Stat(part)
	if err != nil || st.Size() == 0 || st.Size() >= int64(len(big)) {
		t.Fatalf("pausar guarda o .part pela metade: %v %v", st, err)
	}
	if err := h.svc.DownloadGame(dkc.ID); err == nil || !strings.Contains(err.Error(), "pausado") {
		t.Fatalf("pausado não baixa de novo do zero: %v", err)
	}

	// fechou o app: ao abrir, o download volta pausado (não começa sozinho)
	store, err := settings.Open(h.env.Data, h.env.Settings)
	if err != nil {
		t.Fatal(err)
	}
	rec := &recorder{}
	runner := &hooks.Runner{Dir: filepath.Join(h.env.Data, "scripts"), Log: filepath.Join(h.env.Data, "logs", "scripts.log")}
	t.Cleanup(runner.Wait)
	svc, err := core.New(core.Options{Version: "test", Mode: "web", Store: store, Emitter: rec, Hooks: runner, MinPlay: -1,
		Emulators: launcher.Spec{Bin: h.env.Bin, Cores: h.env.Cores}, Bluetooth: bt.NewFake()})
	if err != nil {
		t.Fatal(err)
	}
	if list := svc.Downloads(); len(list) != 1 || list[0].State != "paused" || list[0].Progress < 0.15 {
		t.Fatalf("pausado sobrevive a fechar o app: %+v", list)
	}
	if _, err := svc.SetDownloadOptions(settings.Downloads{}); err != nil {
		t.Fatal(err)
	}
	if err := svc.ResumeDownload(dkc.ID); err != nil {
		t.Fatal(err)
	}
	if ev := rec.waitTask(t, "download:"+dkc.ID); ev.State != "done" {
		t.Fatalf("retomada: %+v", ev)
	}
	got, _ := os.ReadFile(filepath.Join(h.env.Settings.ImportDir, "snes", "Donkey Kong Country (U).zip"))
	if !bytes.Equal(got, big) {
		t.Fatalf("arquivo retomado diferente: %d de %d bytes", len(got), len(big))
	}
	if r := fake.Ranges(); len(r) != 1 || r[0] != "bytes="+strconv.FormatInt(st.Size(), 10)+"-" {
		t.Fatalf("continua de onde parou (Range): %v, tinha %d", r, st.Size())
	}
}

func TestDownloadNightOnly(t *testing.T) {
	h := setup(t)
	fake := fixture.NewFakeRomM()
	connect(t, h, fake)
	dkc, spyro := h.game(t, "Donkey Kong Country"), h.game(t, "Spyro the Dragon")
	day := func() time.Time { return time.Date(2026, 11, 2, 14, 0, 0, 0, time.Local) }
	core.SetDownloadClock(h.svc, day)
	saved, err := h.svc.SetDownloadOptions(settings.Downloads{NightOnly: true, LimitKBps: 5120})
	if err != nil || !saved.Downloads.NightOnly || saved.Downloads.LimitKBps != 5120 {
		t.Fatalf("opções: %+v %v", saved.Downloads, err)
	}

	h.svc.DownloadGame(dkc.ID)
	h.svc.DownloadGame(spyro.ID)
	h.rec.waitTaskEvent(t, "DKC esperando", state(dkc.ID, "waiting"))
	h.rec.waitTaskEvent(t, "Spyro esperando", state(spyro.ID, "waiting"))
	time.Sleep(100 * time.Millisecond)
	for _, d := range h.svc.Downloads() {
		if d.State != "waiting" || !strings.Contains(d.Message, "madrugada") {
			t.Fatalf("de dia, nada baixa: %+v", d)
		}
	}

	// cancelar da fila
	if err := h.svc.CancelTask("download:" + spyro.ID); err != nil {
		t.Fatal(err)
	}
	if ev := h.rec.waitTask(t, "download:"+spyro.ID); ev.State != "cancelled" {
		t.Fatalf("cancelado: %+v", ev)
	}

	// baixar agora mesmo assim
	if err := h.svc.DownloadNow(dkc.ID); err != nil {
		t.Fatal(err)
	}
	if ev := h.rec.waitTask(t, "download:"+dkc.ID); ev.State != "done" {
		t.Fatalf("agora: %+v", ev)
	}

	// a madrugada chega: a fila anda sozinha
	h.svc.DownloadGame(spyro.ID)
	if ev := h.svc.Downloads(); len(ev) != 1 || ev[0].State != "waiting" {
		t.Fatalf("Spyro espera: %+v", ev)
	}
	core.SetDownloadClock(h.svc, func() time.Time { return time.Date(2026, 11, 3, 2, 0, 0, 0, time.Local) })
	h.rec.waitTaskEvent(t, "Spyro baixado de madrugada", state(spyro.ID, "done"))
}
