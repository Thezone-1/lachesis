package main

import (
	"testing"
	"time"
)

func TestParseDarwinWindow(t *testing.T) {
	app, title, err := parseDarwinWindow("Google Chrome\nInbox - Gmail\n")
	if err != nil || app != "google chrome" || title != "Inbox - Gmail" {
		t.Fatalf("got %q %q %v", app, title, err)
	}
	app, title, err = parseDarwinWindow("Finder\n\n")
	if err != nil || app != "finder" || title != "" {
		t.Fatalf("empty title: got %q %q %v", app, title, err)
	}
	if _, _, err = parseDarwinWindow("loginwindow\n\n"); err == nil {
		t.Fatal("login window must count as away")
	}
	if _, _, err = parseDarwinWindow(""); err == nil {
		t.Fatal("empty output must be an error")
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
