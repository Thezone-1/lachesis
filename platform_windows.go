//go:build windows

package main

import (
	"errors"
	"path/filepath"
	"strings"
	"syscall"
	"time"
	"unsafe"
)

var (
	user32   = syscall.NewLazyDLL("user32.dll")
	kernel32 = syscall.NewLazyDLL("kernel32.dll")

	procGetForegroundWindow       = user32.NewProc("GetForegroundWindow")
	procGetWindowTextW            = user32.NewProc("GetWindowTextW")
	procGetWindowThreadProcessId  = user32.NewProc("GetWindowThreadProcessId")
	procGetLastInputInfo          = user32.NewProc("GetLastInputInfo")
	procGetTickCount              = kernel32.NewProc("GetTickCount")
	procOpenProcess               = kernel32.NewProc("OpenProcess")
	procQueryFullProcessImageName = kernel32.NewProc("QueryFullProcessImageNameW")
	procCloseHandle               = kernel32.NewProc("CloseHandle")
)

const processQueryLimitedInformation = 0x1000

type lastInputInfo struct {
	cbSize uint32
	dwTime uint32
}

// ActiveWindow returns the executable name (lowercased, without .exe) and
// window title of the current foreground window.
func ActiveWindow() (app, title string, err error) {
	hwnd, _, _ := procGetForegroundWindow.Call()
	if hwnd == 0 {
		return "", "", errors.New("no foreground window (locked or secure desktop)")
	}

	var titleBuf [512]uint16
	n, _, _ := procGetWindowTextW.Call(hwnd, uintptr(unsafe.Pointer(&titleBuf[0])), uintptr(len(titleBuf)))
	title = syscall.UTF16ToString(titleBuf[:n])

	var pid uint32
	procGetWindowThreadProcessId.Call(hwnd, uintptr(unsafe.Pointer(&pid)))
	if pid == 0 {
		return "", title, errors.New("no pid for foreground window")
	}

	h, _, _ := procOpenProcess.Call(processQueryLimitedInformation, 0, uintptr(pid))
	if h == 0 {
		return "", title, errors.New("cannot open process")
	}
	defer procCloseHandle.Call(h)

	var pathBuf [1024]uint16
	size := uint32(len(pathBuf))
	ok, _, _ := procQueryFullProcessImageName.Call(h, 0, uintptr(unsafe.Pointer(&pathBuf[0])), uintptr(unsafe.Pointer(&size)))
	if ok == 0 {
		return "", title, errors.New("cannot query process image name")
	}
	exe := filepath.Base(syscall.UTF16ToString(pathBuf[:size]))
	app = strings.ToLower(strings.TrimSuffix(exe, filepath.Ext(exe)))
	return app, title, nil
}

// IdleDuration returns how long the user has been without keyboard/mouse input.
func IdleDuration() time.Duration {
	info := lastInputInfo{cbSize: uint32(unsafe.Sizeof(lastInputInfo{}))}
	ok, _, _ := procGetLastInputInfo.Call(uintptr(unsafe.Pointer(&info)))
	if ok == 0 {
		return 0
	}
	tick, _, _ := procGetTickCount.Call()
	// uint32 arithmetic handles tick-count wraparound (~49 days)
	return time.Duration(uint32(tick)-info.dwTime) * time.Millisecond
}

func preflight() error { return nil }

// enableVT turns on ANSI escape processing so the live view works in the
// Windows console. It reports false if the console refuses.
func enableVT() bool {
	getStd := kernel32.NewProc("GetStdHandle")
	getMode := kernel32.NewProc("GetConsoleMode")
	setMode := kernel32.NewProc("SetConsoleMode")
	h, _, _ := getStd.Call(uintptr(^uint32(10))) // STD_OUTPUT_HANDLE
	if h == 0 || h == ^uintptr(0) {
		return false
	}
	var mode uint32
	if ok, _, _ := getMode.Call(h, uintptr(unsafe.Pointer(&mode))); ok == 0 {
		return false
	}
	ok, _, _ := setMode.Call(h, uintptr(mode|0x0004)) // ENABLE_VIRTUAL_TERMINAL_PROCESSING
	return ok != 0
}
