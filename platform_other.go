//go:build !windows

package main

import (
	"errors"
	"time"
)

var errUnsupported = errors.New("window tracking is not supported on this platform yet (windows only)")

// ActiveWindow is unavailable off Windows; reports and tests still run.
func ActiveWindow() (app, title string, err error) { return "", "", errUnsupported }

// IdleDuration is unavailable off Windows.
func IdleDuration() time.Duration { return 0 }
