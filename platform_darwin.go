//go:build darwin

package main

import (
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// The app name comes from lsappinfo, which is fast (about 5 ms) and needs no
// permission. The window title needs System Events and the Accessibility
// permission and costs about 150 ms, so it is cached and refreshed slowly.
const titleScript = `tell application "System Events"
	set p to first application process whose frontmost is true
	return name of front window of p
end tell`

var browsers = map[string]bool{
	"safari": true, "google chrome": true, "arc": true, "firefox": true, "brave browser": true,
	"microsoft edge": true, "opera": true, "vivaldi": true, "orion": true, "zen": true,
}

var titleState struct {
	app, title string
	at         time.Time
	blocked    bool
}

// ActiveWindow returns the frontmost app (lowercased) and its window title.
// Without the Accessibility permission the title is empty and the app is still tracked.
func ActiveWindow() (app, title string, err error) {
	asn, err := exec.Command("lsappinfo", "front").Output()
	if err != nil {
		return "", "", fmt.Errorf("lsappinfo: %w", err)
	}
	out, err := exec.Command("lsappinfo", "info", "-only", "name", strings.TrimSpace(string(asn))).Output()
	if err != nil {
		return "", "", fmt.Errorf("lsappinfo: %w", err)
	}
	app, err = parseLSAppName(string(out))
	if err != nil {
		return "", "", err
	}
	return app, frontTitle(app), nil
}

func frontTitle(app string) string {
	ttl := 30 * time.Second
	if browsers[app] {
		ttl = 10 * time.Second
	}
	if titleState.app == app && time.Since(titleState.at) < ttl {
		return titleState.title
	}
	out, err := exec.Command("osascript", "-e", titleScript).CombinedOutput()
	title := ""
	switch {
	case err == nil:
		title = strings.TrimSpace(string(out))
		titleState.blocked = false
	case isPermissionError(string(out)):
		titleState.blocked = true
	}
	titleState.app, titleState.title, titleState.at = app, title, time.Now()
	return title
}

// titleNote explains, on screen, why window titles are missing.
func titleNote() string {
	if titleState.blocked {
		return "window titles are off: allow Accessibility for your terminal (run `lachesis doctor`)"
	}
	return ""
}

// IdleDuration reads the HID idle time, which needs no permission.
func IdleDuration() time.Duration {
	out, err := exec.Command("ioreg", "-c", "IOHIDSystem").Output()
	if err != nil {
		return 0
	}
	d, err := parseHIDIdle(string(out))
	if err != nil {
		return 0
	}
	return d
}

func preflight() error {
	for _, tool := range []string{"lsappinfo", "ioreg"} {
		if _, err := exec.LookPath(tool); err != nil {
			return fmt.Errorf("%s not found; it ships with macOS, so something is wrong with this system", tool)
		}
	}
	return nil
}

func enableVT() bool { return true }

// platformChecks are the macOS permissions Lachesis relies on.
func platformChecks() []check {
	titleState.at = time.Time{} // force a fresh title lookup
	_, _, err := ActiveWindow()
	checks := []check{{"can read the front app", err == nil || err == errLocked, fmt.Sprint(err)}}
	checks = append(checks, check{
		"can read window titles (Accessibility)", !titleState.blocked,
		"open System Settings > Privacy & Security > Accessibility, switch on your terminal app, then also\n" +
			"    Automation > your terminal > System Events. Tracking works without it, but without titles\n" +
			"    Lachesis cannot tell a YouTube tab from a docs tab. Run `lachesis doctor --open` to jump there.",
	})
	return checks
}

// openPrivacySettings opens the macOS Accessibility pane.
func openPrivacySettings() {
	_ = exec.Command("open", "x-apple.systempreferences:com.apple.preference.security?Privacy_Accessibility").Run()
}
