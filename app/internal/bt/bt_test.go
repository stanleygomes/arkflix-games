package bt

import (
	"context"
	"testing"
)

func TestParseAddress(t *testing.T) {
	ok := map[string]string{
		"9c:aa:1b:00:00:01": "9C:AA:1B:00:00:01",
		"E4:17:D8:00:00:02": "E4:17:D8:00:00:02",
	}
	for in, want := range ok {
		if got, err := ParseAddress(in); err != nil || got != want {
			t.Errorf("%q: %q %v", in, got, err)
		}
	}
	for _, bad := range []string{
		"", "9C:AA:1B:00:00", "9C:AA:1B:00:00:01:02", "9C-AA-1B-00-00-01",
		"9C:AA:1B:00:00:0G", "../../hci0", "9C:AA:1B:00:00:01\n", " 9C:AA:1B:00:00:01",
		"9C:AA:1B:00:00:01; rm -rf /", "9CAA1B000001",
	} {
		if _, err := ParseAddress(bad); err == nil {
			t.Errorf("%q devia ser recusado", bad)
		}
	}
}

func TestIsGamepad(t *testing.T) {
	pads := []string{
		"Xbox Wireless Controller", "DUALSHOCK 4 Wireless Controller", "DualSense Wireless Controller",
		"Wireless Controller", "Pro Controller", "8BitDo Pro 2", "Joy-Con (L)", "Generic Gamepad",
	}
	for _, n := range pads {
		if !IsGamepad(n, "") {
			t.Errorf("%q devia ser controle", n)
		}
	}
	for _, n := range []string{"JBL Flip 5", "Galaxy Buds", "MX Master 3", ""} {
		if IsGamepad(n, "audio-card") {
			t.Errorf("%q não é controle", n)
		}
	}
	if !IsGamepad("Aparelho Qualquer", "input-gaming") {
		t.Error("ícone input-gaming é controle")
	}
}

func TestSort(t *testing.T) {
	ds := []Device{
		{Name: "Fone", Paired: true, Connected: true},
		{Name: "Xbox", Gamepad: true},
		{Name: "8BitDo", Gamepad: true, Paired: true},
		{Name: "DualSense", Gamepad: true, Paired: true, Connected: true},
	}
	Sort(ds)
	want := []string{"DualSense", "8BitDo", "Xbox", "Fone"}
	for i, n := range want {
		if ds[i].Name != n {
			t.Fatalf("ordem: %+v", ds)
		}
	}
}

func TestMacSettingsURL(t *testing.T) {
	if u := macSettingsURL(14); u != "x-apple.systempreferences:com.apple.BluetoothSettings" {
		t.Error(u)
	}
	if u := macSettingsURL(12); u != "x-apple.systempreferences:com.apple.preferences.Bluetooth" {
		t.Error(u)
	}
}

func TestFake(t *testing.T) {
	f := NewFake()
	ds, _ := f.Devices()
	if len(ds) != 1 || ds[0].Address != FakeEightBitDo || !ds[0].Paired {
		t.Fatalf("antes da busca só o 8BitDo pareado: %+v", ds)
	}
	ctx := context.Background()
	if err := f.Pair(ctx, FakeXbox); err == nil {
		t.Fatal("Xbox ainda não foi visto")
	}
	if err := f.StartDiscovery(); err != nil || !f.Adapter().Discovering {
		t.Fatal("busca não começou", err)
	}
	if err := f.Connect(ctx, FakeXbox); err == nil {
		t.Fatal("conectar sem parear devia falhar")
	}
	for _, step := range []error{f.Pair(ctx, FakeXbox), f.SetTrusted(FakeXbox, true), f.Connect(ctx, FakeXbox)} {
		if step != nil {
			t.Fatal(step)
		}
	}
	ds, _ = f.Devices()
	if len(ds) != 2 || ds[1].Address != FakeXbox || !ds[1].Connected || !ds[1].Trusted || ds[1].Battery != 100 {
		t.Fatalf("Xbox pareado: %+v", ds)
	}
	if err := f.Remove(FakeXbox); err != nil {
		t.Fatal(err)
	}
	f.StartDiscovery()
	if ds, _ = f.Devices(); len(ds) != 2 || ds[1].Paired {
		t.Fatalf("esquecido volta não pareado na busca: %+v", ds)
	}
	f.SetPowered(false)
	if ds, _ = f.Devices(); len(ds) != 0 || f.Adapter().Discovering {
		t.Fatal("rádio desligado não lista nem busca")
	}
	if err := f.StartDiscovery(); err == nil {
		t.Fatal("buscar desligado devia falhar")
	}
}

func TestUnsupported(t *testing.T) {
	u := Unsupported{Reason: "x"}
	if a := u.Adapter(); a.Supported || a.Reason != "x" {
		t.Fatal(a)
	}
	if ds, err := u.Devices(); err != nil || ds == nil || len(ds) != 0 {
		t.Fatal("lista vazia (não nil) pro JSON")
	}
	if u.Pair(context.Background(), FakeXbox) != ErrUnsupported {
		t.Fatal("parear sem suporte")
	}
}
