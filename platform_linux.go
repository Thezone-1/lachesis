//go:build linux

package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

// ActiveWindow queries the X11 active window with xprop.
func ActiveWindow() (app, title string, err error) {
	root, err := exec.Command("xprop", "-root", "_NET_ACTIVE_WINDOW").Output()
	if err != nil {
		return "", "", fmt.Errorf("xprop: %w", err)
	}
	id, err := parseXActiveWindow(string(root))
	if err != nil {
		return "", "", errLocked
	}
	props, err := exec.Command("xprop", "-id", id, "_NET_WM_NAME", "WM_CLASS").Output()
	if err != nil {
		return "", "", fmt.Errorf("xprop: %w", err)
	}
	return parseXprop(string(props))
}

// IdleDuration uses xprintidle, which reports milliseconds since the last input.
func IdleDuration() time.Duration {
	out, err := exec.Command("xprintidle").Output()
	if err != nil {
		return 0
	}
	ms, err := strconv.ParseInt(strings.TrimSpace(string(out)), 10, 64)
	if err != nil {
		return 0
	}
	return time.Duration(ms) * time.Millisecond
}

func preflight() error {
	if os.Getenv("XDG_SESSION_TYPE") == "wayland" {
		return errors.New("this is a Wayland session; Linux tracking supports X11 only for now " +
			"(choose an X11/Xorg session at login)")
	}
	if os.Getenv("DISPLAY") == "" {
		return errors.New("no DISPLAY set; run inside a graphical X11 session")
	}
	for _, tool := range []string{"xprop", "xprintidle"} {
		if _, err := exec.LookPath(tool); err != nil {
			return fmt.Errorf("%s not found; install it (Debian/Ubuntu: apt install x11-utils xprintidle)", tool)
		}
	}
	return nil
}

func enableVT() bool { return true }
