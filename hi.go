package main

import (
	"os"
	"strings"
	"time"
)

// bestDeepMinutes is the high score: the most deep-work minutes on any of the
// last 30 recorded days before today.
func bestDeepMinutes(cfg Config, today string) int {
	entries, err := os.ReadDir(dataDir())
	if err != nil {
		return 0
	}
	best, seen := 0, 0
	for i := len(entries) - 1; i >= 0 && seen < 30; i-- {
		name := entries[i].Name()
		day := strings.TrimSuffix(name, ".jsonl")
		if day == name || day == today {
			continue
		}
		seen++
		samples, err := readDay(day)
		if err != nil || len(samples) == 0 {
			continue
		}
		if mins := int(sessionize(samples, cfg.PollSeconds, cfg.Rules).DeepTotal / time.Minute); mins > best {
			best = mins
		}
	}
	return best
}
