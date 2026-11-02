//go:build windows

package updater

import (
	"os/exec"
	"syscall"
)

// detach: grupo próprio e sem janela de console (o app é GUI).
func detach(cmd *exec.Cmd) {
	const createNoWindow, newGroup = 0x08000000, 0x00000200
	cmd.SysProcAttr = &syscall.SysProcAttr{CreationFlags: createNoWindow | newGroup, HideWindow: true}
}
