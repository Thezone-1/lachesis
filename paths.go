package main

import (
	"os"
	"path/filepath"
	"runtime"
)

// baseDir is where config and samples live: %LOCALAPPDATA%\lachesis on Windows,
// ~/Library/Application Support/lachesis on macOS, $XDG_DATA_HOME/lachesis on Linux.
func baseDir() string {
	return filepath.Join(dataRoot(), "lachesis")
}

func dataRoot() string {
	switch runtime.GOOS {
	case "windows":
		if d := os.Getenv("LOCALAPPDATA"); d != "" {
			return d
		}
	case "darwin":
		if h, err := os.UserHomeDir(); err == nil {
			return filepath.Join(h, "Library", "Application Support")
		}
	default:
		if d := os.Getenv("XDG_DATA_HOME"); d != "" {
			return d
		}
		if h, err := os.UserHomeDir(); err == nil {
			return filepath.Join(h, ".local", "share")
		}
	}
	return os.TempDir()
}

// migrateLegacyDir moves the pre-rename data folder ("lacheisis") to the new
// name once, so existing users keep their history.
func migrateLegacyDir() {
	old := filepath.Join(dataRoot(), "lacheisis")
	if _, err := os.Stat(old); err != nil {
		return
	}
	if _, err := os.Stat(baseDir()); err == nil {
		return
	}
	_ = os.Rename(old, baseDir())
}
