//go:build windows

package main

import (
	"os/exec"
	"syscall"
	"unsafe"
)

const stillActive = 259

func processAlive(pid int) bool {
	if pid <= 0 {
		return false
	}
	h, _, _ := procOpenProcess.Call(processQueryLimitedInformation, 0, uintptr(pid))
	if h == 0 {
		return false
	}
	defer procCloseHandle.Call(h)
	var code uint32
	ok, _, _ := kernel32.NewProc("GetExitCodeProcess").Call(h, uintptr(unsafe.Pointer(&code)))
	return ok != 0 && code == stillActive
}

// detach starts the child without a console, outside the terminal's process group.
func detach(cmd *exec.Cmd) {
	const detachedProcess, newProcessGroup = 0x00000008, 0x00000200
	cmd.SysProcAttr = &syscall.SysProcAttr{CreationFlags: detachedProcess | newProcessGroup, HideWindow: true}
}
