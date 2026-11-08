//go:build !windows && !darwin && !linux

package main

import (
	"errors"
	"time"
)

var errUnsupported = errors.New("window tracking is not supported on this platform")

func ActiveWindow() (app, title string, err error) { return "", "", errUnsupported }

func IdleDuration() time.Duration { return 0 }

func preflight() error { return errUnsupported }

func enableVT() bool { return false }

func titleNote() string { return "" }

func platformChecks() []check {
	return []check{{"window tracking on this platform", false, "not supported on this platform"}}
}

func openPrivacySettings() {}
