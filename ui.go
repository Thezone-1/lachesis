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
	session time.Time // when the running session started
	warn    string    // set when the window cannot be read (e.g. a permission is missing)
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

// paintRow colours a row of the scene, one run of equal styles at a time.
func paintRow(o uiOpts, c canvas, y int, tint string) string {
	if !o.color {
		return string(c.r[y])
	}
	var b strings.Builder
	row, st := c.r[y], c.s[y]
	for i := 0; i < len(row); {
		j := i
		for j < len(row) && st[j] == st[i] {
			j++
		}
		seg := string(row[i:j])
		if code := styleCode(st[i], tint); code != "" {
			seg = paint(o, code, seg)
		}
		b.WriteString(seg)
		i = j
	}
	return b.String()
}

func styleCode(st uint8, tint string) string {
	switch st {
	case stThread:
		return "38;5;220" // golden thread
	case stDash:
		return "38;5;240"
	case stBall:
		return tint
	case stObstacle:
		return "1;38;5;196"
	case stScissors:
		return "1;38;5;231"
	case stBanner:
		return "1;38;5;196"
	case stSpark1:
		return "1;38;5;227"
	case stSpark2:
		return "1;38;5;213"
	case stSpark3:
		return "1;38;5;51"
	}
	return ""
}

var titleGradient = []int{213, 177, 141, 105, 69, 33, 39, 45}
var barGradient = []int{196, 202, 208, 214, 220, 226, 190, 154, 118, 82}

// gradientText colours each character of s along the palette.
func gradientText(o uiOpts, s string, palette []int) string {
	if !o.color {
		return s
	}
	var b strings.Builder
	n := 0
	for _, ch := range s {
		if ch == ' ' {
			b.WriteRune(ch)
			continue
		}
		b.WriteString(paint(o, fmt.Sprintf("1;38;5;%d", palette[n%len(palette)]), string(ch)))
		n++
	}
	return b.String()
}

// coloredBar draws the score bar with a red-to-green gradient.
func coloredBar(o uiOpts, total int) string {
	filled := total / 10
	if filled > 10 {
		filled = 10
	}
	on, off := "█", "░"
	if o.ascii {
		on, off = "#", "-"
	}
	var b strings.Builder
	for i := 0; i < 10; i++ {
		if i < filled {
			b.WriteString(paint(o, fmt.Sprintf("38;5;%d", barGradient[i]), on))
		} else {
			b.WriteString(paint(o, "38;5;238", off))
		}
	}
	return b.String()
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
		return "38;5;244"
	case c == Productive:
		return "1;38;5;82"
	case c == Distracting:
		return "1;38;5;203"
	default:
		return "1;38;5;214"
	}
}

// renderLive returns the frame as lines, without a trailing newline.
func renderLive(l live, frame int, r *runner, o uiOpts) []string {
	if o.width < minWidth {
		o.width = minWidth
	}
	scene := r.drawCanvas(o.width, l.idle)
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
	hiText, todayText := fmt.Sprintf("HI %05d", hi), fmt.Sprintf("%05d", l.deep)
	title0 := "L A C H E S I S"
	pad := max(2, o.width-len(title0)-len(hiText)-2-len(todayText))
	lines := []string{
		"  " + gradientText(o, title0, titleGradient) + strings.Repeat(" ", pad) +
			paint(o, "38;5;220", hiText) + "  " + paint(o, "1;38;5;231", todayText),
	}
	for i := 0; i < sceneRows; i++ {
		lines = append(lines, "  "+paintRow(o, scene, i, tint))
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
		"  "+paint(o, "1;38;5;82", beat+" tracking")+paint(o, "90", sessionLabel(l)+fmt.Sprintf(" · %d %s · last %s", l.samples, plural(l.samples, "sample", "samples"), ago)),
		fmt.Sprintf("  now    %s  %s", paint(o, tint, catTag(l.cat)+" "+state), now),
		fmt.Sprintf("  today  %s active · %s idle · focus block %s", liveDur(l.m.Active), liveDur(l.m.Idle), liveDur(l.block)),
	)
	if l.m.Active < minScoreActive {
		lines = append(lines, fmt.Sprintf("  score  %s --  warming up the thread (needs %s of activity)", scoreBar(0, o.ascii), fmtDur(minScoreActive)))
	} else {
		lines = append(lines, fmt.Sprintf("  %s  %s %s  %s", paint(o, "38;5;244", "score"), coloredBar(o, l.score.Total), paint(o, "1", fmt.Sprint(l.score.Total)), verdict(l.score.Total)))
	}
	switch {
	case l.warn != "":
		lines = append(lines, "  "+paint(o, "33", "! "+l.warn))
	case r.cutting() && r.cutF >= 14:
		lines = append(lines, "  "+paint(o, "31", "✂ "+banter(r.cutApp, l.m.Active)))
	case r.celebrating():
		lines = append(lines, "  "+paint(o, "1;38;5;227", "✦ deep work: a 25-minute block of unbroken thread"))
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

func sessionLabel(l live) string {
	if l.session.IsZero() || l.now.IsZero() {
		return ""
	}
	return " · session " + liveDur(l.now.Sub(l.session))
}
