package main

import (
	"fmt"
	"sort"
	"strings"
	"time"
)

type appStat struct {
	app   string
	dur   time.Duration
	cat   Category
	votes map[Category]time.Duration
	tops  map[string]time.Duration // title -> time, for drill-down
}

// buildReport aggregates a day's samples into per-app active time.
// Each sample represents pollSeconds of attention. An app's category tag is
// the category its time majority fell into (titles can override the app).
func buildReport(samples []Sample, pollSeconds int, rules []Rule) (stats []appStat, active, idle time.Duration) {
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
			st = &appStat{app: s.App, tops: map[string]time.Duration{}, votes: map[Category]time.Duration{}}
			byApp[s.App] = st
		}
		st.dur += tick
		st.tops[s.Title] += tick
		st.votes[classifySample(s, rules)] += tick
	}
	for _, st := range byApp {
		st.cat = Neutral
		var best time.Duration
		for _, cat := range []Category{Productive, Distracting, Neutral} {
			if st.votes[cat] > best {
				st.cat, best = cat, st.votes[cat]
			}
		}
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

func catTag(c Category) string {
	switch c {
	case Productive:
		return "[+]"
	case Distracting:
		return "[!]"
	default:
		return "[·]"
	}
}

func printReport(day string, cfg Config) error {
	samples, err := readDay(day)
	if err != nil {
		return err
	}
	if len(samples) == 0 {
		fmt.Printf("Lachesis — %s\nNo thread measured. Is `lachesis track` running?\n", day)
		return nil
	}

	m := sessionize(samples, cfg.PollSeconds, cfg.Rules)
	orphans := attributeCommits(m.Blocks, dayCommits(cfg.Repos, day))
	score := computeScore(m, cfg.Weights)

	fmt.Printf("Lachesis — %s\n", day)
	if m.Active < minScoreActive {
		fmt.Printf("Score -- — warming up (%s of activity so far, a score needs %s)\n", fmtDur(m.Active), fmtDur(minScoreActive))
	} else {
		fmt.Printf("Score %d — %s\n", score.Total, verdict(score.Total))
	}
	fmt.Printf("  focus %.2f · depth %.2f · cohesion %.2f · yield %.2f\n",
		score.Focus, score.Depth, score.Cohesion, score.Yield)
	fmt.Printf("Thread measured: %s active · %s idle\n", fmtDur(m.Active), fmtDur(m.Idle))
	fmt.Printf("  [+] productive %s · [·] neutral %s · [!] distracting %s\n",
		fmtDur(m.Productive), fmtDur(m.Neutral), fmtDur(m.Distracting))
	if yl := yieldLine(m.Blocks, m.DeepTotal); yl != "" {
		fmt.Printf("  %s\n", yl)
	}
	fmt.Println()

	stats, _, _ := buildReport(samples, cfg.PollSeconds, cfg.Rules)
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
		fmt.Printf("  %s %-22s %7s  %s%s\n", catTag(st.cat), st.app, fmtDur(st.dur), bar, note)
	}

	if m.DeepCount > 0 {
		fmt.Printf("\nDeep work — %d block(s), %s total, longest %s, %.0f switches/h:\n",
			m.DeepCount, fmtDur(m.DeepTotal), fmtDur(m.Longest), m.SwitchesPerHour())
		for _, b := range m.Blocks {
			if !b.Deep() {
				continue
			}
			fmt.Printf("  %s–%s  %-18s %7s", b.Start.Format("15:04"), b.End.Format("15:04"), b.App, fmtDur(b.Dur))
			for i, c := range b.Commits {
				sep := "  ● "
				if i > 0 {
					sep = " · "
				}
				fmt.Printf("%s%.7s %s (%s)", sep, c.Hash, c.Subject, c.Repo)
			}
			fmt.Println()
		}
	}
	if len(orphans) > 0 {
		fmt.Printf("\nCommits outside deep work (%d):\n", len(orphans))
		for _, c := range orphans {
			fmt.Printf("  %s  %.7s %s (%s)\n", c.TS.Format("15:04"), c.Hash, c.Subject, c.Repo)
		}
	}

	for _, c := range cuts {
		fmt.Printf("\n✂ Atropos: %s\n", c)
	}
	for _, w := range farmerWarnings(m.Blocks) {
		fmt.Printf("\n⚠ %s\n", w)
	}
	return nil
}
