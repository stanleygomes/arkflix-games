package gamepad

import (
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"unsafe"
)

// Native: o Go lê os controles (evdev). Fora do Linux, a interface lê.
const Native = true

func ioc(dir, typ, nr, size uintptr) uintptr { return dir<<30 | size<<16 | typ<<8 | nr }

func ioctl(fd uintptr, req uintptr, arg unsafe.Pointer) error {
	if _, _, e := syscall.Syscall(syscall.SYS_IOCTL, fd, req, uintptr(arg)); e != 0 {
		return e
	}
	return nil
}

func testBit(bits []byte, n int) bool { return n/8 < len(bits) && bits[n/8]&(1<<(n%8)) != 0 }

type devInfo struct {
	name     string
	vendor   uint16
	gamepad  bool
	joystick bool
	abs      map[uint16]absInfo
}

func probe(f *os.File) (devInfo, error) {
	fd := f.Fd()
	var info devInfo
	name := make([]byte, 256)
	if err := ioctl(fd, ioc(2, 'E', 0x06, 256), unsafe.Pointer(&name[0])); err != nil {
		return info, err
	}
	info.name = strings.TrimRight(string(name), "\x00")
	id := make([]uint16, 4) // bustype, vendor, product, version
	if err := ioctl(fd, ioc(2, 'E', 0x02, 8), unsafe.Pointer(&id[0])); err == nil {
		info.vendor = id[1]
	}
	keys := make([]byte, 0x300/8)
	if err := ioctl(fd, ioc(2, 'E', 0x20+evKey, uintptr(len(keys))), unsafe.Pointer(&keys[0])); err != nil {
		return info, err
	}
	info.gamepad = testBit(keys, btnSouth)
	info.joystick = !info.gamepad && testBit(keys, btnJoystick)
	absBits := make([]byte, 8)
	info.abs = map[uint16]absInfo{}
	if err := ioctl(fd, ioc(2, 'E', 0x20+evAbs, uintptr(len(absBits))), unsafe.Pointer(&absBits[0])); err == nil {
		for code := range uint16(0x20) {
			if testBit(absBits, int(code)) {
				var ai absInfo
				if ioctl(fd, ioc(2, 'E', 0x40+uintptr(code), unsafe.Sizeof(ai)), unsafe.Pointer(&ai)) == nil {
					info.abs[code] = ai
				}
			}
		}
	}
	return info, nil
}

func (m *Manager) scan() {
	paths, _ := filepath.Glob(filepath.Join(m.Dir, "event*"))
	changed := false
	for _, path := range paths {
		m.mu.Lock()
		_, known := m.files[path]
		m.mu.Unlock()
		if known {
			continue
		}
		f, err := os.Open(path)
		if err != nil {
			continue // sem permissão: não é controle do usuário
		}
		info, err := probe(f)
		if err != nil || (!info.gamepad && !info.joystick) {
			f.Close()
			continue
		}
		m.mu.Lock()
		pad := &Pad{Index: m.nextIndex(), Name: info.name, Family: Family(info.vendor, info.name), Path: path}
		m.pads[path], m.files[path] = pad, f
		m.mu.Unlock()
		changed = true
		dec := newDecoder(*pad, info.joystick, info.abs, m.publish)
		go func() {
			dec.decodeStream(f) // termina quando o controle é desconectado
			m.mu.Lock()
			delete(m.pads, path)
			delete(m.files, path)
			subs := append([]func([]Pad){}, m.onPads...)
			m.mu.Unlock()
			f.Close()
			pads := m.Pads()
			for _, fn := range subs {
				fn(pads)
			}
		}()
	}
	if changed {
		m.mu.Lock()
		subs := append([]func([]Pad){}, m.onPads...)
		m.mu.Unlock()
		pads := m.Pads()
		for _, fn := range subs {
			fn(pads)
		}
	}
}

// Describe é usado em logs.
