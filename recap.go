package main

import (
	"fmt"
	"os"
	"strings"
	"time"
)

const recapWidth = 54

// sessionSamples gathers the samples recorded between start and end.
func sessionSamples(start, end time.Time) []Sample {
	var out []Sample
	for d := time.Date(start.Year(), start.Month(), start.Day(), 0, 0, 0, 0, start.Location()); !d.After(end); d = d.AddDate(0, 0, 1) {
		day, _ := readDay(d.Format("2006-01-02"))
		for _, s := range day {
			if !s.TS.Before(start) && !s.TS.After(end) {
				out = append(out, s)
			}
		}
	}
	return out
}

// sessionRecap is the card shown when a session ends.
func sessionRecap(start, end time.Time, label string, cfg Config, o uiOpts) []string {
	samples := sessionSamples(start, end)
	m := sessionize(samples, cfg.PollSeconds, cfg.Rules)
	score := computeScore(m, cfg.Weights)

	commits := 0
	if len(samples) > 0 {
		var all []Commit
		for d := start; !d.After(end); d = d.AddDate(0, 0, 1) {
			for _, c := range dayCommits(cfg.Repos, d.Format("2006-01-02")) {
				if !c.TS.Before(start) && !c.TS.After(end) {
					all = append(all, c)
				}
			}
		}
		attributeCommits(m.Blocks, all)
		for _, b := range m.Blocks {
			commits += len(b.Commits)
		}
	}

	title := "session complete"
	if label != "" {
		title += fmt.Sprintf(" · %q", label)
	}
	type row struct{ plain, painted string }
	rows := []row{{"✦ " + title, paint(o, "1;38;5;227", "✦ "+title)}}
	add := func(s string) { rows = append(rows, row{s, s}) }

	add(fmt.Sprintf("%s on the clock · %s active", liveDur(end.Sub(start)), liveDur(m.Active)))
	if m.Active < minScoreActive {
		add(fmt.Sprintf("too short to score (a score needs %s of activity)", fmtDur(minScoreActive)))
	} else {
		add(fmt.Sprintf("productive %s · distracting %s", liveDur(m.Productive), liveDur(m.Distracting)))
		if m.DeepCount > 0 {
			add(fmt.Sprintf("%d deep block%s · longest %s", m.DeepCount, map[bool]string{true: "", false: "s"}[m.DeepCount == 1], liveDur(m.Longest)))
		} else {
			add("no deep block yet (25 minutes of unbroken focus)")
		}
		if commits > 0 {
			add(fmt.Sprintf("%d commit%s linked to deep work", commits, map[bool]string{true: "", false: "s"}[commits == 1]))
		}
		plain := fmt.Sprintf("score %s %d  %s", scoreBar(score.Total, o.ascii), score.Total, verdict(score.Total))
		rows = append(rows, row{plain, fmt.Sprintf("%s %s %s  %s", paint(o, "38;5;244", "score"), coloredBar(o, score.Total), paint(o, "1", fmt.Sprint(score.Total)), verdict(score.Total))})
	}

	h, v, tl, tr, bl, br := "─", "│", "╭", "╮", "╰", "╯"
	if o.ascii {
		h, v, tl, tr, bl, br = "-", "|", "+", "+", "+", "+"
	}
	edge := strings.Repeat(h, recapWidth+2)
	lines := []string{"  " + paint(o, "38;5;220", tl+edge+tr)}
	for _, r := range rows {
		pad := max(0, recapWidth-len([]rune(r.plain)))
		lines = append(lines, "  "+paint(o, "38;5;220", v)+" "+r.painted+strings.Repeat(" ", pad)+" "+paint(o, "38;5;220", v))
	}
	lines = append(lines, "  "+paint(o, "38;5;220", bl+edge+br))
	return lines
}

func printRecap(start, end time.Time, label string, cfg Config) {
	o := uiOpts{ascii: os.Getenv("LACHESIS_ASCII") != "", color: isTerminal(os.Stdout) && os.Getenv("NO_COLOR") == ""}
	fmt.Println()
	for _, l := range sessionRecap(start, end, label, cfg, o) {
		fmt.Println(l)
	}
	fmt.Println()
}
