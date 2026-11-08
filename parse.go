package main

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var errLocked = errors.New("no foreground window (locked screen or login window)")

var lsNameRe = regexp.MustCompile(`^"([^"]*)"`)

// parseLSAppName reads the app name from `lsappinfo info -only name <asn>`.
func parseLSAppName(out string) (string, error) {
	m := lsNameRe.FindStringSubmatch(strings.TrimSpace(out))
	if m == nil {
		return "", errors.New("no application name")
	}
	name := strings.ToLower(strings.TrimSpace(m[1]))
	if name == "" || name == "loginwindow" {
		return "", errLocked
	}
	return name, nil
}

// isPermissionError reports whether osascript failed because macOS withheld
// Automation (-1743) or Accessibility (-25211) access, as opposed to an app
// simply having no window.
func isPermissionError(out string) bool {
	return strings.Contains(out, "-1743") || strings.Contains(out, "-25211") ||
		strings.Contains(out, "not allowed assistive") || strings.Contains(out, "not authorized")
}

var hidIdleRe = regexp.MustCompile(`"HIDIdleTime"\s*=\s*(\d+)`)

// parseHIDIdle extracts the idle time in nanoseconds from `ioreg -c IOHIDSystem`.
func parseHIDIdle(out string) (time.Duration, error) {
	m := hidIdleRe.FindStringSubmatch(out)
	if m == nil {
		return 0, errors.New("HIDIdleTime not found")
	}
	ns, err := strconv.ParseInt(m[1], 10, 64)
	if err != nil {
		return 0, err
	}
	return time.Duration(ns), nil
}

var xActiveRe = regexp.MustCompile(`window id # (0x[0-9a-fA-F]+)`)

// parseXActiveWindow returns the window id from `xprop -root _NET_ACTIVE_WINDOW`.
func parseXActiveWindow(out string) (string, error) {
	m := xActiveRe.FindStringSubmatch(out)
	if m == nil || m[1] == "0x0" {
		return "", errors.New("no active window")
	}
	return m[1], nil
}

var (
	xTitleRe = regexp.MustCompile(`(?m)^_NET_WM_NAME\([^)]*\) = "(.*)"$`)
	xClassRe = regexp.MustCompile(`(?m)^WM_CLASS\([^)]*\) = "(.*?)", "(.*?)"$`)
)

// parseXprop reads the title and application class from
// `xprop -id <id> _NET_WM_NAME WM_CLASS`.
func parseXprop(out string) (app, title string, err error) {
	if m := xTitleRe.FindStringSubmatch(out); m != nil {
		title = strings.NewReplacer(`\"`, `"`, `\\`, `\`).Replace(m[1])
	}
	if m := xClassRe.FindStringSubmatch(out); m != nil {
		app = strings.ToLower(m[2])
	}
	if app == "" {
		return "", title, fmt.Errorf("no WM_CLASS in xprop output")
	}
	return app, title, nil
}
