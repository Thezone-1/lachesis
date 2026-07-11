package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

type Config struct {
	PollSeconds int               `json:"poll_seconds"`
	IdleSeconds int               `json:"idle_seconds"`
	Allotments  map[string]string `json:"allotments"` // app -> duration ("2h", "90m")
}

func defaultConfig() Config {
	return Config{PollSeconds: 5, IdleSeconds: 60, Allotments: map[string]string{}}
}

func baseDir() string {
	return filepath.Join(os.Getenv("LOCALAPPDATA"), "lacheisis")
}

func configPath() string {
	return filepath.Join(baseDir(), "config.json")
}

func loadConfig() Config {
	cfg := defaultConfig()
	data, err := os.ReadFile(configPath())
	if err != nil {
		return cfg
	}
	json.Unmarshal(data, &cfg)
	if cfg.PollSeconds <= 0 {
		cfg.PollSeconds = 5
	}
	if cfg.IdleSeconds <= 0 {
		cfg.IdleSeconds = 60
	}
	if cfg.Allotments == nil {
		cfg.Allotments = map[string]string{}
	}
	return cfg
}

func saveConfig(cfg Config) error {
	if err := os.MkdirAll(baseDir(), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(configPath(), data, 0o644)
}

// allotmentFor returns the parsed daily allotment for an app, or 0 if none set.
func (c Config) allotmentFor(app string) time.Duration {
	s, ok := c.Allotments[app]
	if !ok {
		return 0
	}
	d, err := time.ParseDuration(s)
	if err != nil {
		fmt.Fprintf(os.Stderr, "warning: bad allotment %q for %s\n", s, app)
		return 0
	}
	return d
}
