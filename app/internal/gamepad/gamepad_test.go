package gamepad

import (
	"bytes"
	"testing"
)

func collect(joystick bool, abs map[uint16]absInfo, raw ...[]byte) []Event {
	var out []Event
	d := newDecoder(Pad{Index: 0, Name: "Teste", Family: "xbox"}, joystick, abs, func(e Event) { out = append(out, e) })
	d.decodeStream(bytes.NewReader(bytes.Join(raw, nil)))
	return out
}

func TestButtons(t *testing.T) {
	ev := collect(false, nil,
		Raw(evKey, 0x130, 1), Raw(evKey, 0x130, 2), Raw(evKey, 0x130, 0),
		Raw(evKey, 0x13b, 1), Raw(evKey, 0x999, 1),
	)
	want := []Event{
		{Button: "south", Pressed: true}, {Button: "south", Pressed: false}, {Button: "start", Pressed: true},
	}
	if len(ev) != len(want) {
		t.Fatalf("eventos: %+v", ev)
	}
	for i, w := range want {
		if ev[i].Button != w.Button || ev[i].Pressed != w.Pressed || ev[i].Family != "xbox" {
			t.Errorf("%d: got %+v want %+v", i, ev[i], w)
		}
	}
}

func TestHatAndStick(t *testing.T) {
	abs := map[uint16]absInfo{absX: {Min: -32768, Max: 32767}, absY: {Min: 0, Max: 255}}
	ev := collect(false, abs,
		Raw(evAbs, absHat0Y, -1), Raw(evAbs, absHat0Y, 0),
		Raw(evAbs, absX, 30000), // passou do limite: right
		Raw(evAbs, absX, 15000), // entre 0.35 e 0.6: continua apertado (histerese)
		Raw(evAbs, absX, 1000),  // voltou: solta
		Raw(evAbs, absY, 0),     // eixo 0..255 no mínimo: up
	)
	got := []string{}
	for _, e := range ev {
		state := "-"
		if e.Pressed {
			state = "+"
		}
		got = append(got, state+e.Button+e.Source)
	}
	want := []string{"+up", "-up", "+rightstick", "-rightstick", "+upstick"}
	if len(got) != len(want) {
		t.Fatalf("got %v want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got %v want %v", got, want)
		}
	}
}

func TestTriggers(t *testing.T) {
	abs := map[uint16]absInfo{absZ: {Min: 0, Max: 1023}}
	ev := collect(false, abs, Raw(evAbs, absZ, 900), Raw(evAbs, absZ, 400), Raw(evAbs, absZ, 100))
	if len(ev) != 2 || ev[0].Button != "l2" || !ev[0].Pressed || ev[1].Pressed {
		t.Fatalf("gatilho: %+v", ev)
	}
}

func TestGenericJoystick(t *testing.T) {
	ev := collect(true, nil, Raw(evKey, 0x129, 1), Raw(evAbs, absZ, 255))
	if len(ev) != 1 || ev[0].Button != "start" {
		t.Fatalf("joystick genérico: %+v", ev)
	}
}

func TestFamily(t *testing.T) {
	cases := []struct {
		vendor uint16
		name   string
		want   string
	}{
		{0x045e, "Controller", "xbox"},
		{0x054c, "Wireless Controller", "playstation"},
		{0x2dc8, "Pro 2", "8bitdo"},
		{0, "8BitDo SN30 Pro", "8bitdo"},
		{0, "DualSense Wireless Controller", "playstation"},
		{0x1234, "USB Gamepad", "generic"},
	}
	for _, c := range cases {
		if got := Family(c.vendor, c.name); got != c.want {
			t.Errorf("Family(%#x, %q) = %s, want %s", c.vendor, c.name, got, c.want)
		}
	}
}

func TestManagerInjectAndEmptyDir(t *testing.T) {
	m := New(t.TempDir())
	var got []Event
	m.Subscribe(func(e Event) { got = append(got, e) })
	m.Start()
	defer m.Close()
	m.Inject(Event{Button: "east", Pressed: true})
	if len(got) != 1 || len(m.Pads()) != 0 {
		t.Fatalf("inject: %+v pads %+v", got, m.Pads())
	}
}
