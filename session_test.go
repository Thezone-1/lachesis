package main

import (
	"fmt"
	"os"
	"testing"
	"time"
)

func tempData(t *testing.T) {
	t.Helper()
	root := t.TempDir()
	t.Setenv("LOCALAPPDATA", root)
	t.Setenv("XDG_DATA_HOME", root)
	t.Setenv("HOME", root)
}

func writeSessionFile(t *testing.T, info sessionInfo) {
	t.Helper()
	data := fmt.Sprintf(`{"pid":%d,"start":%q}`, info.PID, info.Start.Format(time.RFC3339))
	if err := os.WriteFile(pidPath(), []byte(data), 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestProcessAlive(t *testing.T) {
	if !processAlive(os.Getpid()) {
		t.Fatal("this process is alive")
	}
	if processAlive(0) || processAlive(-5) {
		t.Fatal("non-positive pids are never alive")
	}
}

func TestAcquireAndRelease(t *testing.T) {
	tempData(t)
	info, err := acquireSession("deep work")
	if err != nil {
		t.Fatal(err)
	}
	cur, ok := runningSession()
	if !ok || cur.PID != os.Getpid() || cur.Label != "deep work" {
		t.Fatalf("session not registered: %+v %v", cur, ok)
	}
	releaseSession(info)
	if _, ok := runningSession(); ok {
		t.Fatal("session should be gone after release")
	}
	if _, _, label, ok := lastSession(); !ok || label != "deep work" {
		t.Fatal("the finished session should be in the log")
	}
}

func TestSecondSessionIsRefused(t *testing.T) {
	tempData(t)
	if err := os.MkdirAll(baseDir(), 0o700); err != nil {
		t.Fatal(err)
	}
	// the parent process stands in for "another live tracker"
	writeSessionFile(t, sessionInfo{PID: os.Getppid(), Start: time.Now()})
	if _, err := acquireSession(""); err == nil {
		t.Fatal("must refuse while another tracker is alive")
	}
}

func TestStalePidFileIsReplaced(t *testing.T) {
	tempData(t)
	if err := os.MkdirAll(baseDir(), 0o700); err != nil {
		t.Fatal(err)
	}
	writeSessionFile(t, sessionInfo{PID: 2147483000, Start: time.Now()})
	if _, err := acquireSession(""); err != nil {
		t.Fatalf("a dead tracker's pid file must not block a new session: %v", err)
	}
}

func TestStopRequestFlag(t *testing.T) {
	tempData(t)
	if err := os.MkdirAll(baseDir(), 0o700); err != nil {
		t.Fatal(err)
	}
	if stopRequested() {
		t.Fatal("no request yet")
	}
	if err := os.WriteFile(stopPath(), nil, 0o600); err != nil {
		t.Fatal(err)
	}
	if !stopRequested() {
		t.Fatal("request should be seen")
	}
	if _, err := acquireSession(""); err != nil || stopRequested() {
		t.Fatal("a new session must clear an old stop request")
	}
}

func TestFlagValue(t *testing.T) {
	if flagValue([]string{"--plain", "--label", "writing"}, "--label") != "writing" {
		t.Fatal("flag value not found")
	}
	if flagValue([]string{"--label"}, "--label") != "" {
		t.Fatal("a missing value should be empty")
	}
}
