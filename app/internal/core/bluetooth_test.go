package core_test

import (
	"testing"

	"github.com/stanleygomes/arkflix-games/app/internal/bt"
	"github.com/stanleygomes/arkflix-games/app/internal/core"
)

func (r *recorder) lastBT() (core.BTInfo, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for i := len(r.events) - 1; i >= 0; i-- {
		if info, ok := r.data[i].(core.BTInfo); ok && r.events[i] == "bluetooth" {
			return info, true
		}
	}
	return core.BTInfo{}, false
}

func find(info core.BTInfo, addr string) (bt.Device, bool) {
	for _, d := range info.Devices {
		if d.Address == addr {
			return d, true
		}
	}
	return bt.Device{}, false
}

func TestBluetoothPair(t *testing.T) {
	h := setup(t)
	st := h.svc.BluetoothStatus()
	if !st.Supported || !st.Powered || st.Scanning || len(st.Devices) != 1 {
		t.Fatalf("início: %+v", st)
	}
	if err := h.svc.BluetoothPair("nada"); err == nil {
		t.Fatal("endereço inválido")
	}
	if err := h.svc.BluetoothPair(bt.FakeXbox); err == nil {
		t.Fatal("Xbox só aparece depois da busca")
	}
	if err := h.svc.BluetoothScan(5); err != nil {
		t.Fatal(err)
	}
	info, ok := h.rec.lastBT()
	if !ok || !info.Scanning {
		t.Fatalf("evento da busca: %+v", info)
	}
	if d, ok := find(info, bt.FakeXbox); !ok || d.Paired || !d.Gamepad {
		t.Fatalf("Xbox na busca: %+v", info.Devices)
	}
	if err := h.svc.BluetoothPair("9c:aa:1b:00:00:01"); err != nil { // minúsculas valem
		t.Fatal(err)
	}
	if ev := h.rec.waitTask(t, "bt:"+bt.FakeXbox); ev.State != "done" {
		t.Fatalf("parear: %+v", ev)
	}
	st = h.svc.BluetoothStatus()
	if d, _ := find(st, bt.FakeXbox); !d.Paired || !d.Trusted || !d.Connected {
		t.Fatalf("pareado, confiável e conectado: %+v", d)
	}
	if st.Scanning {
		t.Fatal("parear encerra a busca")
	}
	if err := h.svc.BluetoothDisconnect(bt.FakeXbox); err != nil {
		t.Fatal(err)
	}
	if err := h.svc.BluetoothConnect(bt.FakeXbox); err != nil {
		t.Fatal(err)
	}
	if err := h.svc.BluetoothForget(bt.FakeXbox); err != nil {
		t.Fatal(err)
	}
	if _, ok := find(h.svc.BluetoothStatus(), bt.FakeXbox); ok {
		t.Fatal("esquecido some da lista")
	}
	if err := h.svc.BluetoothPower(false); err != nil {
		t.Fatal(err)
	}
	if err := h.svc.BluetoothScan(0); err == nil {
		t.Fatal("buscar com o rádio desligado")
	}
}
