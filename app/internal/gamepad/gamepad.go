// Package gamepad lê controles direto do kernel no Linux (evdev,
// /dev/input/event*), sem cgo: o WebKitGTK do Ubuntu não tem a Gamepad API,
// então é o Go que entrega os botões pra interface, com nomes posicionais
// padrão. No Windows e no macOS o WebView tem a Gamepad API e quem lê o
// controle é a interface (Native = false).
package gamepad

import (
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
	"sync"
	"time"
)

// Event é um botão apertado ou solto. Button usa os nomes de input.Physical
// (south, east, up, l1...). Direções do analógico esquerdo saem como up/down/
// left/right com Source "stick".
type Event struct {
	Pad     int    `json:"pad"`
	Name    string `json:"name"`
	Family  string `json:"family"`
	Button  string `json:"button"`
	Pressed bool   `json:"pressed"`
	Source  string `json:"source,omitempty"`
}

// Pad é um controle conectado.
type Pad struct {
	Index  int    `json:"index"`
	Name   string `json:"name"`
	Family string `json:"family"` // xbox, playstation, 8bitdo, nintendo, generic
	Path   string `json:"-"`
}

// Códigos do kernel (linux/input-event-codes.h).
const (
	evKey = 0x01
	evAbs = 0x03

	btnJoystick = 0x120
	btnSouth    = 0x130

	absX, absY, absZ, absRX, absRY, absRZ = 0x00, 0x01, 0x02, 0x03, 0x04, 0x05
	absGas, absBrake                      = 0x09, 0x0a
	absHat0X, absHat0Y                    = 0x10, 0x11
)

var gamepadKeys = map[uint16]string{
	0x130: "south", 0x131: "east", 0x133: "north", 0x134: "west",
	0x136: "l1", 0x137: "r1", 0x138: "l2", 0x139: "r2",
	0x13a: "select", 0x13b: "start", 0x13c: "home", 0x13d: "l3", 0x13e: "r3",
	0x220: "up", 0x221: "down", 0x222: "left", 0x223: "right",
}

// Joysticks HID genéricos (pads USB baratos estilo SNES) usam BTN_TRIGGER...
var joystickKeys = map[uint16]string{
	0x120: "north", 0x121: "east", 0x122: "south", 0x123: "west",
	0x124: "l1", 0x125: "r1", 0x126: "l2", 0x127: "r2",
	0x128: "select", 0x129: "start",
}

// Family identifica o fabricante pelo vendor ID USB ou pelo nome.
func Family(vendor uint16, name string) string {
	switch vendor {
	case 0x045e:
		return "xbox"
	case 0x054c:
		return "playstation"
	case 0x2dc8:
		return "8bitdo"
	case 0x057e:
		return "nintendo"
	}
	lower := strings.ToLower(name)
	switch {
	case strings.Contains(lower, "xbox") || strings.Contains(lower, "x-box"):
		return "xbox"
	case strings.Contains(lower, "playstation") || strings.Contains(lower, "dualsense") || strings.Contains(lower, "dualshock") || strings.Contains(lower, "sony"):
		return "playstation"
	case strings.Contains(lower, "8bitdo"):
		return "8bitdo"
	case strings.Contains(lower, "nintendo") || strings.Contains(lower, "pro controller"):
		return "nintendo"
	}
	return "generic"
}

type absInfo struct {
	Value, Min, Max, Fuzz, Flat, Resolution int32
}

// decoder transforma eventos crus em Event, com histerese no analógico.
type decoder struct {
	pad    Pad
	keys   map[uint16]string
	abs    map[uint16]absInfo
	held   map[string]bool
	hatX   int32
	hatY   int32
	isJoy  bool
	output func(Event)
}

func newDecoder(p Pad, joystick bool, abs map[uint16]absInfo, out func(Event)) *decoder {
	keys := gamepadKeys
	if joystick {
		keys = joystickKeys
	}
	return &decoder{pad: p, keys: keys, abs: abs, held: map[string]bool{}, isJoy: joystick, output: out}
}

func (d *decoder) emit(button string, pressed bool, source string) {
	if d.held[button+source] == pressed {
		return
	}
	d.held[button+source] = pressed
	d.output(Event{Pad: d.pad.Index, Name: d.pad.Name, Family: d.pad.Family, Button: button, Pressed: pressed, Source: source})
}

// norm leva o valor do eixo pra -1..1 (ou 0..1 em gatilho).
func (d *decoder) norm(code uint16, v int32) float64 {
	info, ok := d.abs[code]
	if !ok || info.Max == info.Min {
		info = absInfo{Min: -32768, Max: 32767}
	}
	mid := float64(info.Min+info.Max) / 2
	half := float64(info.Max-info.Min) / 2
	return (float64(v) - mid) / half
}

func (d *decoder) axis(code uint16, v int32, neg, pos, source string) {
	x := d.norm(code, v)
	press, release := 0.6, 0.35 // histerese: não pisca na borda
	switch {
	case x <= -press:
		d.emit(neg, true, source)
	case x > -release:
		d.emit(neg, false, source)
	}
	switch {
	case x >= press:
		d.emit(pos, true, source)
	case x < release:
		d.emit(pos, false, source)
	}
}

func (d *decoder) trigger(code uint16, v int32, button string) {
	info, ok := d.abs[code]
	if !ok || info.Max == info.Min {
		info = absInfo{Min: 0, Max: 255}
	}
	x := float64(v-info.Min) / float64(info.Max-info.Min)
	switch {
	case x >= 0.5:
		d.emit(button, true, "")
	case x < 0.3:
		d.emit(button, false, "")
	}
}

func (d *decoder) hat(code uint16, v int32) {
	if code == absHat0X {
		d.hatX = v
		d.emit("left", v < 0, "")
		d.emit("right", v > 0, "")
	} else {
		d.hatY = v
		d.emit("up", v < 0, "")
		d.emit("down", v > 0, "")
	}
}

func (d *decoder) handle(typ, code uint16, value int32) {
	switch typ {
	case evKey:
		if name, ok := d.keys[code]; ok && value != 2 { // 2 = autorepeat
			d.emit(name, value == 1, "")
		}
	case evAbs:
		switch code {
		case absHat0X, absHat0Y:
			d.hat(code, value)
		case absX:
			d.axis(code, value, "left", "right", "stick")
		case absY:
			d.axis(code, value, "up", "down", "stick")
		case absZ, absBrake:
			if !d.isJoy {
				d.trigger(code, value, "l2")
			}
		case absRZ, absGas:
			if !d.isJoy {
				d.trigger(code, value, "r2")
			}
		}
	}
}

const eventSize = 24 // struct input_event em 64 bits: timeval(16) + type + code + value

// decodeStream lê eventos crus até erro/EOF.
func (d *decoder) decodeStream(r io.Reader) error {
	buf := make([]byte, eventSize*64)
	for {
		n, err := r.Read(buf)
		for off := 0; off+eventSize <= n; off += eventSize {
			e := buf[off : off+eventSize]
			d.handle(binary.LittleEndian.Uint16(e[16:18]), binary.LittleEndian.Uint16(e[18:20]), int32(binary.LittleEndian.Uint32(e[20:24])))
		}
		if err != nil {
			return err
		}
	}
}

// Raw monta um evento cru (pros testes).
func Raw(typ, code uint16, value int32) []byte {
	b := make([]byte, eventSize)
	binary.LittleEndian.PutUint16(b[16:], typ)
	binary.LittleEndian.PutUint16(b[18:], code)
	binary.LittleEndian.PutUint32(b[20:], uint32(value))
	return b
}

// ioctl do evdev.
// Manager acha os controles, lê os eventos e avisa quem assinou.
type Manager struct {
	Dir string // /dev/input

	mu      sync.Mutex
	pads    map[string]*Pad
	files   map[string]*os.File
	subs    []func(Event)
	onPads  []func([]Pad)
	stop    chan struct{}
	stopped bool
}

// New cria o gerente; Start começa a procurar controles.
func New(dir string) *Manager {
	if dir == "" {
		dir = "/dev/input"
	}
	return &Manager{Dir: dir, pads: map[string]*Pad{}, files: map[string]*os.File{}, stop: make(chan struct{})}
}

// Subscribe recebe cada botão.
func (m *Manager) Subscribe(fn func(Event)) {
	m.mu.Lock()
	m.subs = append(m.subs, fn)
	m.mu.Unlock()
}

// OnPadsChanged é chamado quando um controle entra ou sai.
func (m *Manager) OnPadsChanged(fn func([]Pad)) {
	m.mu.Lock()
	m.onPads = append(m.onPads, fn)
	m.mu.Unlock()
}

// Inject entrega um evento como se viesse do controle (modo de teste).
func (m *Manager) Inject(e Event) { m.publish(e) }

func (m *Manager) publish(e Event) {
	m.mu.Lock()
	subs := append([]func(Event){}, m.subs...)
	m.mu.Unlock()
	for _, fn := range subs {
		fn(e)
	}
}

// Pads lista os controles conectados.
func (m *Manager) Pads() []Pad {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]Pad, 0, len(m.pads))
	for _, p := range m.pads {
		out = append(out, *p)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Index < out[j].Index })
	return out
}

// Start procura controles agora e a cada 2 s (conectar depois funciona).
func (m *Manager) Start() {
	m.scan()
	go func() {
		t := time.NewTicker(2 * time.Second)
		defer t.Stop()
		for {
			select {
			case <-m.stop:
				return
			case <-t.C:
				m.scan()
			}
		}
	}()
}

// Close para tudo.
func (m *Manager) Close() {
	m.mu.Lock()
	if !m.stopped {
		m.stopped = true
		close(m.stop)
		for _, f := range m.files {
			f.Close()
		}
	}
	m.mu.Unlock()
}

func (m *Manager) nextIndex() int {
	used := map[int]bool{}
	for _, p := range m.pads {
		used[p.Index] = true
	}
	for i := 0; ; i++ {
		if !used[i] {
			return i
		}
	}
}

func (p Pad) String() string { return fmt.Sprintf("#%d %s (%s)", p.Index, p.Name, p.Family) }
