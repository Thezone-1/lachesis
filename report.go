package main

import (
	"fmt"
	"sort"
	"strings"
	"time"
)

type appStat struct {
	app  string
	dur  time.Duration
	tops map[string]time.Duration // title -> time, for drill-down
}

// buildReport aggregates a day's samples into per-app active time.
// Each sample represents pollSeconds of attention.
func buildReport(samples []Sample, pollSeconds int) (stats []appStat, active, idle time.Duration) {
	tick := time.Duration(pollSeconds) * time.Second
	byApp := map[string]*appStat{}
	for _, s := range samples {
		if s.Idle || s.App == "" {
			idle += tick
			continue
		}
		active += tick
		st, ok := byApp[s.App]
		if !ok {
			st = &appStat{app: s.App, tops: map[string]time.Duration{}}
			byApp[s.App] = st
		}
		st.dur += tick
		st.tops[s.Title] += tick
	}
	for _, st := range byApp {
		stats = append(stats, *st)
	}
	sort.Slice(stats, func(i, j int) bool { return stats[i].dur > stats[j].dur })
	return stats, active, idle
}

func fmtDur(d time.Duration) string {
	d = d.Round(time.Minute)
	h := d / time.Hour
	m := (d % time.Hour) / time.Minute
	if h > 0 {
		return fmt.Sprintf("%dh%02dm", h, m)
	}
	return fmt.Sprintf("%dm", m)
}

func printReport(day string, cfg Config) error {
	samples, err := readDay(day)
	if err != nil {
		return err
	}
	if len(samples) == 0 {
		fmt.Printf("Lacheisis — %s\nNo thread measured. Is `lacheisis track` running?\n", day)
		return nil
	}

	stats, active, idle := buildReport(samples, cfg.PollSeconds)

	fmt.Printf("Lacheisis — %s\n", day)
	fmt.Printf("Thread measured: %s active · %s idle\n\n", fmtDur(active), fmtDur(idle))

	maxDur := stats[0].dur
	var cuts []string
	for _, st := range stats {
		if st.dur < time.Minute {
			continue
		}
		bar := strings.Repeat("█", int(1+19*st.dur/max(maxDur, 1)))
		allot := cfg.allotmentFor(st.app)
		note := ""
		if allot > 0 {
			pct := 100 * st.dur / allot
			note = fmt.Sprintf("  (%d%% of %s allotted)", pct, fmtDur(allot))
			if st.dur > allot {
				cuts = append(cuts, fmt.Sprintf("%s exceeded its allotment (%s / %s)", st.app, fmtDur(st.dur), fmtDur(allot)))
			}
		}
		fmt.Printf("  %-22s %7s  %s%s\n", st.app, fmtDur(st.dur), bar, note)
	}

	for _, c := range cuts {
		fmt.Printf("\n✂ Atropos: %s\n", c)
	}
	return nil
}
