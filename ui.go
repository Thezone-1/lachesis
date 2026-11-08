package main

import (
	"fmt"
	"strings"
	"time"
)

// uiOpts controls how the live view draws.
type uiOpts struct {
	ascii      bool // plain ASCII glyphs instead of box drawing and block characters
	color      bool
	width      int  // total width of the thread strip's inside, in cells
	hideTitles bool // for screen recordings: show the app, never the window title
}

// live is everything the view needs to draw one frame.
type live struct {
	sample  Sample
	cat     Category
	m       DayMetrics
	score   Score
	cut     string // app that blew its allotment, if any
	idle    bool
	block   time.Duration // length of the current focus block
	hi      int           // best deep-work minutes on any earlier day
	deep    int           // deep-work minutes today
	samples int
	last    time.Time
	now     time.Time
	warn    string // set when the window cannot be read (e.g. a permission is missing)
}

// minScoreActive is how much activity a day needs before a score means anything.
const minScoreActive = 5 * time.Minute

func scoreBar(total int, ascii bool) string {
	filled := total / 10
	if filled > 10 {
		filled = 10
	}
	on, off := "█", "░"
	if ascii {
		on, off = "#", "-"
	}
	return strings.Repeat(on, filled) + strings.Repeat(off, 10-filled)
}

func liveDur(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	return fmtDur(d)
}

func trunc(s string, n int) string {
	r := []rune(s)
	if n <= 1 || len(r) <= n {
		return s
	}
	return string(r[:n-1]) + "…"
}

func paint(o uiOpts, code, s string) string {
	if !o.color {
		return s
	}
	return "\x1b[" + code + "m" + s + "\x1b[0m"
}

func catColor(c Category, idle bool) string {
	switch {
	case idle:
		return "90"
	case c == Productive:
		return "32"
	case c == Distracting:
		return "31"
	default:
		return "33"
	}
}

// renderLive returns the frame as lines, without a trailing newline.
func renderLive(l live, frame int, r *runner, o uiOpts) []string {
	if o.width < minWidth {
		o.width = minWidth
	}
	scene := r.draw(o.width, l.cat, l.idle)
	tint := catColor(l.cat, l.idle)

	app := l.sample.App
	state := "away"
	if !l.idle {
		state = string(l.cat)
	}
	if app == "" {
		app = "-"
	}
	title := l.sample.Title
	if o.hideTitles {
		title = ""
	}
	now := trunc(strings.TrimSpace(fmt.Sprintf("%s %s", app, title)), o.width-14)

	hi := l.hi
	if l.deep > hi {
		hi = l.deep
	}
	scores := fmt.Sprintf("HI %05d  %05d", hi, l.deep)
	title0 := "L A C H E S I S"
	pad := max(2, o.width-len(title0)-len(scores))
	lines := []string{
		"  " + paint(o, "1", title0) + strings.Repeat(" ", pad) + paint(o, "90", scores),
	}
	for i := 0; i < sceneRows; i++ {
		colour := "90"
		if i == groundRow {
			colour = tint
		}
		lines = append(lines, "  "+paint(o, colour, scene[i]))
	}

	beat := "●"
	if o.ascii {
		beat = "*"
	}
	if frame%8 >= 5 {
		beat = " "
	}
	ago := "-"
	if !l.last.IsZero() && !l.now.IsZero() {
		ago = fmt.Sprintf("%ds ago", int(l.now.Sub(l.last).Seconds()))
	}
	lines = append(lines,
		"  "+paint(o, "32", beat+" tracking")+paint(o, "90", fmt.Sprintf(" · %d %s · last %s", l.samples, plural(l.samples, "sample", "samples"), ago)),
		fmt.Sprintf("  now    %s  %s", paint(o, tint, catTag(l.cat)+" "+state), now),
		fmt.Sprintf("  today  %s active · %s idle · focus block %s", liveDur(l.m.Active), liveDur(l.m.Idle), liveDur(l.block)),
	)
	if l.m.Active < minScoreActive {
		lines = append(lines, fmt.Sprintf("  score  %s --  warming up the thread (needs %s of activity)", scoreBar(0, o.ascii), fmtDur(minScoreActive)))
	} else {
		lines = append(lines, fmt.Sprintf("  score  %s %d  %s", paint(o, tint, scoreBar(l.score.Total, o.ascii)), l.score.Total, verdict(l.score.Total)))
	}
	switch {
	case l.warn != "":
		lines = append(lines, "  "+paint(o, "33", "! "+l.warn))
	case r.cutting() && r.cutF >= 14:
		lines = append(lines, "  "+paint(o, "31", "✂ "+banter(r.cutApp, l.m.Active)))
	case l.cut != "":
		lines = append(lines, "  "+paint(o, "31", "✂ Atropos: "+l.cut+" is over its allotment"))
	default:
		lines = append(lines, "")
	}
	lines = append(lines, "  "+paint(o, "90", "Ctrl-C pauses the thread"))
	return lines
}

func plural(n int, one, many string) string {
	if n == 1 {
		return one
	}
	return many
}
