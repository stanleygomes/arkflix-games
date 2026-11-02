package core

import (
	"path/filepath"
	"syscall"
	"unsafe"
)

var getDiskFreeSpaceEx = syscall.NewLazyDLL("kernel32.dll").NewProc("GetDiskFreeSpaceExW")

// freeSpace: bytes livres no disco da pasta (-1 se não deu pra saber).
func freeSpace(dir string) int64 {
	for d := dir; ; d = filepath.Dir(d) {
		p, err := syscall.UTF16PtrFromString(d)
		if err == nil {
			var free uint64
			if r, _, _ := getDiskFreeSpaceEx.Call(uintptr(unsafe.Pointer(p)), uintptr(unsafe.Pointer(&free)), 0, 0); r != 0 {
				return int64(free)
			}
		}
		if d == filepath.Dir(d) {
			return -1
		}
	}
}
