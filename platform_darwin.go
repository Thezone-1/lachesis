//go:build darwin

package main

import (
	"fmt"
	"os/exec"
	"time"
)

const frontWindowScript = `tell application "System Events"
	set p to first application process whose frontmost is true
	set n to name of p
	set t to ""
	try
		set t to name of front window of p
	end try
	return n & linefeed & t
end tell`

// ActiveWindow returns the frontmost app (lowercased) and its window title.
// The title needs the Accessibility permission; without it the title is empty.
func ActiveWindow() (app, title string, err error) {
	out, err := exec.Command("osascript", "-e", frontWindowScript).Output()
	if err != nil {
		return "", "", fmt.Errorf("osascript: %w", err)
	}
	return parseDarwinWindow(string(out))
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

// preflight fails early, with instructions, if macOS blocks the window query.
func preflight() error {
	if _, _, err := ActiveWindow(); err != nil && err != errLocked {
		return fmt.Errorf("%w\n\nmacOS blocked the query. Open System Settings > Privacy & Security and allow\n"+
			"your terminal under Automation (System Events) and Accessibility, then run again", err)
	}
	return nil
}
