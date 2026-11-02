package core

import (
	"errors"
	"path/filepath"
	"strings"
	"syscall"
	"unsafe"
)

var getDiskFreeSpaceEx = syscall.NewLazyDLL("kernel32.dll").NewProc("GetDiskFreeSpaceExW")

// diskUsage: tamanho total e livre (pra você) do disco da pasta; -1 se não
// deu pra saber. Sobe até achar uma pasta que existe.
func diskUsage(dir string) (total, free int64) {
	for d := dir; ; d = filepath.Dir(d) {
		p, err := syscall.UTF16PtrFromString(d)
		if err == nil {
			var avail, size uint64
			if r, _, _ := getDiskFreeSpaceEx.Call(uintptr(unsafe.Pointer(p)), uintptr(unsafe.Pointer(&avail)), uintptr(unsafe.Pointer(&size)), 0); r != 0 {
				return int64(size), int64(avail)
			}
		}
		if d == filepath.Dir(d) {
			return -1, -1
		}
	}
}

func freeSpace(dir string) int64 { _, free := diskUsage(dir); return free }

// sameDisk: mesma letra de unidade (ou mesmo compartilhamento).
func sameDisk(a, b string) bool {
	va, vb := filepath.VolumeName(a), filepath.VolumeName(b)
	return va != "" && strings.EqualFold(va, vb)
}

// crossDevice: ERROR_NOT_SAME_DEVICE (o MoveFile não atravessa unidades).
func crossDevice(err error) bool { return errors.Is(err, syscall.Errno(17)) }
