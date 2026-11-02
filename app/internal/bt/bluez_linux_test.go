package bt

import (
	"errors"
	"testing"

	"github.com/godbus/dbus/v5"
)

func v(x any) dbus.Variant { return dbus.MakeVariant(x) }

func TestDevicesFrom(t *testing.T) {
	objs := managed{
		"/org/bluez/hci0": {ifAdapter: {"Powered": v(true), "Discovering": v(false)}},
		"/org/bluez/hci0/dev_9C_AA_1B_00_00_01": {
			ifDevice: {"Address": v("9c:aa:1b:00:00:01"), "Name": v("Xbox Wireless Controller"), "Icon": v("input-gaming")},
		},
		"/org/bluez/hci0/dev_E4_17_D8_00_00_02": {
			ifDevice:  {"Address": v("E4:17:D8:00:00:02"), "Name": v("8BitDo Pro 2"), "Paired": v(true), "Connected": v(true), "Trusted": v(true)},
			ifBattery: {"Percentage": v(byte(77))},
		},
		// beacon BLE sem nome: fora
		"/org/bluez/hci0/dev_11_22_33_44_55_66": {ifDevice: {"Address": v("11:22:33:44:55:66"), "Alias": v("11-22-33-44-55-66")}},
		// pareado sem Name: usa o Alias
		"/org/bluez/hci0/dev_AA_BB_CC_DD_EE_FF": {ifDevice: {"Address": v("AA:BB:CC:DD:EE:FF"), "Alias": v("Fone"), "Bonded": v(true)}},
		// endereço estranho nunca passa
		"/org/bluez/hci0/dev_x": {ifDevice: {"Address": v("zz"), "Name": v("x")}},
		// de outro adaptador (desligado): fora
		"/org/bluez/hci1":                       {ifAdapter: {"Powered": v(false)}},
		"/org/bluez/hci1/dev_01_02_03_04_05_06": {ifDevice: {"Address": v("01:02:03:04:05:06"), "Name": v("Outro")}},
	}
	ds := devicesFrom(objs)
	if len(ds) != 3 {
		t.Fatalf("3 aparelhos: %+v", ds)
	}
	eight, xbox, fone := ds[0], ds[1], ds[2]
	if eight.Name != "8BitDo Pro 2" || eight.Battery != 77 || !eight.Gamepad || !eight.Paired || !eight.Trusted {
		t.Errorf("8BitDo: %+v", eight)
	}
	if xbox.Address != "9C:AA:1B:00:00:01" || xbox.Paired || !xbox.Gamepad || xbox.Battery != -1 {
		t.Errorf("Xbox: %+v", xbox)
	}
	if fone.Name != "Fone" || !fone.Paired || fone.Gamepad {
		t.Errorf("fone: %+v", fone)
	}
}

func TestAdapterPath(t *testing.T) {
	if _, ok := adapterPath(managed{"/": {}}); ok {
		t.Fatal("sem adaptador")
	}
	p, _ := adapterPath(managed{
		"/org/bluez/hci1": {ifAdapter: {"Powered": v(false)}},
		"/org/bluez/hci0": {ifAdapter: {"Powered": v(false)}},
	})
	if p != "/org/bluez/hci0" {
		t.Fatal(p)
	}
}

func TestTranslate(t *testing.T) {
	if translate(nil) != nil {
		t.Fatal("nil")
	}
	if !errors.Is(translate(dbus.Error{Name: "org.bluez.Error.AlreadyExists"}), errExists) {
		t.Fatal("AlreadyExists")
	}
	if !errors.Is(translate(&dbus.Error{Name: "org.bluez.Error.NotReady"}), errOff) {
		t.Fatal("NotReady (ponteiro)")
	}
	if err := translate(dbus.Error{Name: "org.bluez.Error.Whatever", Body: []any{"x"}}); err == nil || err.Error() != "o Bluetooth falhou: x" {
		t.Fatal(err)
	}
}
