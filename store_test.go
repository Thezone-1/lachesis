package main

import (
	"os"
	"testing"
	"time"
)

func TestReadDaySkipsTornLines(t *testing.T) {
	root := t.TempDir()
	t.Setenv("LOCALAPPDATA", root)
	t.Setenv("XDG_DATA_HOME", root)
	t.Setenv("HOME", root)

	ts := time.Date(2026, 7, 12, 9, 0, 0, 0, time.Local)
	if err := appendSample(Sample{TS: ts, App: "code", Title: "main.go"}); err != nil {
		t.Fatal(err)
	}
	f, err := os.OpenFile(dayFile("2026-07-12"), os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		t.Fatal(err)
	}
	f.WriteString("{\"ts\":\"2026-07-12T09:00:05")
	f.Close()

	got, err := readDay("2026-07-12")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].App != "code" {
		t.Fatalf("want 1 clean sample, got %+v", got)
	}
}

func TestReadDayMissingFileIsEmpty(t *testing.T) {
	root := t.TempDir()
	t.Setenv("LOCALAPPDATA", root)
	t.Setenv("XDG_DATA_HOME", root)
	t.Setenv("HOME", root)

	got, err := readDay("2001-01-01")
	if err != nil || got != nil {
		t.Fatalf("want nil,nil got %v,%v", got, err)
	}
}
