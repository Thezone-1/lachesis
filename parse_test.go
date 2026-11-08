package main

import (
	"testing"
	"time"
)

func TestParseLSAppName(t *testing.T) {
	app, err := parseLSAppName(`"Google Chrome" ASN:0x0-0x1b01b: (in front)` + "\n  bundleID=[ NULL ]\n")
	if err != nil || app != "google chrome" {
		t.Fatalf("got %q %v", app, err)
	}
	if _, err = parseLSAppName(`"loginwindow" ASN:0x0-0x1: (in front)`); err == nil {
		t.Fatal("login window must count as away")
	}
	if _, err = parseLSAppName("garbage"); err == nil {
		t.Fatal("unparseable output must be an error")
	}
}

func TestIsPermissionError(t *testing.T) {
	if !isPermissionError("execution error: Not authorized to send Apple events to System Events. (-1743)") {
		t.Fatal("-1743 is a permission problem")
	}
	if !isPermissionError("osascript is not allowed assistive access. (-25211)") {
		t.Fatal("-25211 is a permission problem")
	}
	if isPermissionError("execution error: Can't get window 1 of process. Invalid index. (-1719)") {
		t.Fatal("an app with no window is not a permission problem")
	}
}

func TestParseHIDIdle(t *testing.T) {
	out := `    | |   "HIDIdleTime" = 1234567890` + "\n"
	d, err := parseHIDIdle(out)
	if err != nil || d != 1234567890*time.Nanosecond {
		t.Fatalf("got %v %v", d, err)
	}
	if _, err = parseHIDIdle("nothing here"); err == nil {
		t.Fatal("missing key must be an error")
	}
}

func TestParseXActiveWindow(t *testing.T) {
	id, err := parseXActiveWindow("_NET_ACTIVE_WINDOW(WINDOW): window id # 0x4a00007\n")
	if err != nil || id != "0x4a00007" {
		t.Fatalf("got %q %v", id, err)
	}
	if _, err = parseXActiveWindow("_NET_ACTIVE_WINDOW(WINDOW): window id # 0x0\n"); err == nil {
		t.Fatal("0x0 means no active window")
	}
}

func TestParseXprop(t *testing.T) {
	out := "_NET_WM_NAME(UTF8_STRING) = \"main.go - lachesis - Visual Studio Code\"\n" +
		"WM_CLASS(STRING) = \"code\", \"Code\"\n"
	app, title, err := parseXprop(out)
	if err != nil || app != "code" || title != "main.go - lachesis - Visual Studio Code" {
		t.Fatalf("got %q %q %v", app, title, err)
	}
	if _, _, err = parseXprop("_NET_WM_NAME(UTF8_STRING) = \"x\"\n"); err == nil {
		t.Fatal("missing class must be an error")
	}
}
