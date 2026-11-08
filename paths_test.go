package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestMigrateLegacyDir(t *testing.T) {
	root := t.TempDir()
	t.Setenv("LOCALAPPDATA", root)
	t.Setenv("XDG_DATA_HOME", root)
	t.Setenv("HOME", root)

	old := filepath.Join(dataRoot(), "lacheisis")
	if err := os.MkdirAll(old, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(old, "config.json"), []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}

	migrateLegacyDir()

	if _, err := os.Stat(filepath.Join(baseDir(), "config.json")); err != nil {
		t.Fatalf("config not migrated: %v", err)
	}
	if _, err := os.Stat(old); !os.IsNotExist(err) {
		t.Fatalf("legacy dir should be gone")
	}
}

func TestMigrateKeepsExistingNewDir(t *testing.T) {
	root := t.TempDir()
	t.Setenv("LOCALAPPDATA", root)
	t.Setenv("XDG_DATA_HOME", root)
	t.Setenv("HOME", root)

	if err := os.MkdirAll(filepath.Join(dataRoot(), "lacheisis"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(baseDir(), 0o755); err != nil {
		t.Fatal(err)
	}
	migrateLegacyDir()
	if _, err := os.Stat(filepath.Join(dataRoot(), "lacheisis")); err != nil {
		t.Fatalf("legacy dir must be left alone when the new one exists")
	}
}
