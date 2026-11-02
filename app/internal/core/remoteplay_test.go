package core_test

import (
	"strings"
	"testing"

	"github.com/stanleygomes/arkflix-games/app/internal/core"
	"github.com/stanleygomes/arkflix-games/app/internal/remoteplay"
)

func (r *recorder) lastRemotePlay() (core.RemotePlayInfo, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for i := len(r.events) - 1; i >= 0; i-- {
		if info, ok := r.data[i].(core.RemotePlayInfo); ok && r.events[i] == "remoteplay" {
			return info, true
		}
	}
	return core.RemotePlayInfo{}, false
}

// Com o Sunshine falso: instala (tarefa), liga, mostra o IP e desliga.
func TestRemotePlayFixture(t *testing.T) {
	h := setup(t)
	st := h.svc.RemotePlayStatus()
	if !st.Supported || !st.AutoInstall || st.Installed || st.Running || st.Busy || st.LanIP != remoteplay.FakeIP {
		t.Fatalf("início: %+v", st)
	}
	if err := h.svc.StartRemotePlay(); err == nil {
		t.Fatal("ligar sem instalar")
	}
	if err := h.svc.InstallRemotePlay(); err != nil {
		t.Fatal(err)
	}
	if err := h.svc.InstallRemotePlay(); err == nil || !strings.Contains(err.Error(), "já está") {
		t.Fatalf("duas ao mesmo tempo: %v", err)
	}
	if ev := h.rec.waitTask(t, "remoteplay"); ev.State != "done" || ev.Message != "Sunshine instalado" {
		t.Fatalf("fim: %+v", ev)
	}
	if info, ok := h.rec.lastRemotePlay(); !ok || !info.Installed {
		t.Fatalf("evento depois de instalar: %+v", info)
	}
	if err := h.svc.StartRemotePlay(); err != nil {
		t.Fatal(err)
	}
	if info, _ := h.rec.lastRemotePlay(); !info.Running || !info.Started {
		t.Fatalf("evento ligado: %+v", info)
	}
	// abrir a página pede o app desktop (no modo web a interface abre o link)
	if err := h.svc.OpenRemotePlayPage(); err == nil {
		t.Fatal("abrir página sem janela")
	}
	if err := h.svc.StopRemotePlay(); err != nil {
		t.Fatal(err)
	}
	if st := h.svc.RemotePlayStatus(); st.Running {
		t.Fatalf("desligado: %+v", st)
	}
	if err := h.svc.StopRemotePlay(); err == nil {
		t.Fatal("desligar duas vezes")
	}
}
