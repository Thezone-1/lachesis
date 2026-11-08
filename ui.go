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
	sample Sample
	cat    Category
	m      DayMetrics
	score  Score
	cut    string // app that blew its allotment, if any
	idle   bool
	block  time.Duration // length of the current focus block
}

// minScoreActive is how much activity a day needs before a score means anything.
const minScoreActive = 5 * time.Minute

const threadPattern = "~~~~~o~~~~~~~o~~~~~~"

// threadStep says whether the thread advances on this frame: fast when
// productive, slow when neutral, stuttering when distracted, frozen when away.
func threadStep(frame int, cat Category, idle bool) bool {
	switch {
	case idle:
		return false
	case cat == Productive:
		return true
	case cat == Distracting:
		return frame%2 == 0
	default:
		return frame%3 == 0
	}
}

// threadLine draws the scrolling thread. A distracted thread frays, and an
// exceeded allotment cuts it with scissors.
func threadLine(w, pos int, cat Category, idle, cut, ascii bool) string {
	b := []rune(strings.Repeat(" ", w))
	for i := range b {
		c := rune(threadPattern[(i+pos)%len(threadPattern)])
		if idle {
			c = '.'
		} else if cat == Distracting && (i+pos)%5 == 0 {
			c = ','
		}
		b[i] = c
	}
	if cut && w > 12 {
		mid := w / 2
		scissors := '✂'
		if ascii {
			scissors = 'x'
		}
		b[mid] = scissors
		for i := mid + 1; i < mid+4 && i < w; i++ {
			b[i] = ' '
		}
	}
	return string(b)
}

func yarn(frame int, ascii, idle bool) string {
	if ascii {
		return string("|/-\\"[(frame/2)%4])
	}
	if idle {
		return "◌"
	}
	return string([]rune("◐◓◑◒")[(frame/2)%4])
}

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
func renderLive(l live, frame, pos int, o uiOpts) []string {
	if o.width < 20 {
		o.width = 20
	}
	h, v, tl, tr, bl, br := "─", "│", "╭", "╮", "╰", "╯"
	if o.ascii {
		h, v, tl, tr, bl, br = "-", "|", "+", "+", "+", "+"
	}
	edge := strings.Repeat(h, o.width+4)
	thread := threadLine(o.width-1, pos, l.cat, l.idle, l.cut != "", o.ascii)
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

	lines := []string{
		"  " + paint(o, "1", "L A C H E S I S") + "  " + paint(o, "90", "version "+version),
		"  " + tl + edge + tr,
		"  " + v + " " + paint(o, tint, yarn(frame, o.ascii, l.idle)) + " " + paint(o, tint, thread) + " " + v,
		"  " + bl + edge + br,
		fmt.Sprintf("  now    %s  %s", paint(o, tint, catTag(l.cat)+" "+state), now),
		fmt.Sprintf("  today  %s active · %s idle · focus block %s", liveDur(l.m.Active), liveDur(l.m.Idle), liveDur(l.block)),
	}
	if l.m.Active < minScoreActive {
		lines = append(lines,
			fmt.Sprintf("  score  %s --  warming up the thread (needs %s of activity)", scoreBar(0, o.ascii), fmtDur(minScoreActive)),
			"")
	} else {
		lines = append(lines,
			fmt.Sprintf("  score  %s %d  %s", paint(o, tint, scoreBar(l.score.Total, o.ascii)), l.score.Total, verdict(l.score.Total)),
			fmt.Sprintf("         focus %.2f · depth %.2f · cohesion %.2f · yield %.2f", l.score.Focus, l.score.Depth, l.score.Cohesion, l.score.Yield))
	}
	if l.cut != "" {
		lines = append(lines, "  "+paint(o, "31", "✂ Atropos: "+l.cut+" is over its allotment"))
	} else {
		lines = append(lines, "")
	}
	lines = append(lines, "  "+paint(o, "90", "Ctrl-C pauses the thread"))
	return lines
}
