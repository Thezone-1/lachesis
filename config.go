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
	Allotments  map[string]string `json:"allotments"`      // app -> duration ("2h", "90m")
	Rules       []Rule            `json:"rules"`           // classification rules, title rules before app rules
	Repos       []string          `json:"repos"`           // git repos scanned for commit yield
	Weights     Weights           `json:"weights"`         // score component weights, must sum to 100
	Theme       string            `json:"theme,omitempty"` // "mono" (default) or "color"
}

// Weights are the score components; each component is a 0–1 ratio.
type Weights struct {
	Focus    int `json:"focus"`    // productive / active
	Depth    int `json:"depth"`    // deep-work / productive
	Cohesion int `json:"cohesion"` // 1 - switches-per-hour penalty
	Yield    int `json:"yield"`    // artifact-linked deep hours / deep hours
}

func defaultWeights() Weights {
	return Weights{Focus: 35, Depth: 25, Cohesion: 15, Yield: 25}
}

func defaultConfig() Config {
	return Config{
		PollSeconds: 5,
		IdleSeconds: 60,
		Allotments:  map[string]string{},
		Rules:       defaultRules(),
		Weights:     defaultWeights(),
	}
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
	if cfg.Rules == nil {
		cfg.Rules = defaultRules()
	}
	if cfg.Weights == (Weights{}) {
		cfg.Weights = defaultWeights()
	}
	return cfg
}

func saveConfig(cfg Config) error {
	if err := os.MkdirAll(baseDir(), 0o700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(configPath(), data, 0o600)
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
