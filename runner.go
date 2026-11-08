package main

import (
	"fmt"
	"strings"
	"time"
)

const (
	sceneRows   = 5 // three rows of sky, the ball row, and the thread
	ballRow     = 3
	groundRow   = 4
	ballX       = 10
	obsW        = 3
	jumpSpan    = 10 // cells scrolled during one jump
	cutFrames   = 48 // length of the scissors scene, in frames
	celebFrames = 28 // length of the sparkle celebration, in frames
	cutX        = ballX + 12
	minWidth    = 30
)

// jumpHeight is how many rows above the ground the ball is after each cell of a jump.
var jumpHeight = [jumpSpan]int{1, 2, 2, 2, 2, 2, 2, 2, 2, 1}

var (
	scissorsOpen = [3]string{
		` o o `,
		`  X  `,
		` / \ `,
	}
	scissorsShut = [3]string{
		` o o `,
		`  X  `,
		`  |  `,
	}
)

// runner is the little world: a ball of yarn rolling along the thread, jumping
// the distractions that come at it, until Atropos drops her scissors.
type runner struct {
	scroll    int   // cells travelled, drives the dashes and the roll
	obs       []int // left edge of each obstacle
	untilNext int   // cells until the next obstacle appears
	jump      int   // cells into the current jump, -1 when on the ground
	bump      int   // frames left of the stumble after hitting an obstacle
	cutF      int   // frame of the scissors scene, -1 when not cutting
	celeb     int   // frame of the sparkle celebration, -1 when none
	cutApp    string
}

func newRunner() *runner {
	return &runner{jump: -1, cutF: -1, celeb: -1, untilNext: 18}
}

// cellsThisFrame is how far the world scrolls: fast when productive, slow when
// distracted, nothing when away.
func cellsThisFrame(frame int, cat Category, idle bool) int {
	switch {
	case idle:
		return 0
	case cat == Productive:
		return 1
	case cat == Distracting:
		if frame%3 == 0 {
			return 1
		}
		return 0
	default:
		if frame%2 == 0 {
			return 1
		}
		return 0
	}
}

func (r *runner) startCut(app string) {
	if r.cutF < 0 {
		r.cutF, r.cutApp = 0, app
	}
}

func (r *runner) cutting() bool { return r.cutF >= 0 }

// startCelebrate plays the sparkles for a completed deep-work block.
func (r *runner) startCelebrate() {
	if r.celeb < 0 {
		r.celeb = 0
	}
}

func (r *runner) celebrating() bool { return r.celeb >= 0 }

// step advances the world by one frame.
func (r *runner) step(frame, width int, cat Category, idle bool) {
	if r.celeb >= 0 {
		if r.celeb++; r.celeb >= celebFrames {
			r.celeb = -1
		}
	}
	if r.cutF >= 0 {
		r.cutF++
		if r.cutF >= cutFrames {
			*r = runner{jump: -1, cutF: -1, celeb: -1, untilNext: 18, scroll: r.scroll}
		}
		return
	}
	if r.bump > 0 {
		r.bump--
		return
	}
	if cellsThisFrame(frame, cat, idle) == 0 {
		return
	}
	r.scroll++
	keep := r.obs[:0]
	for _, o := range r.obs {
		if o-1 > -obsW {
			keep = append(keep, o-1)
		}
	}
	r.obs = keep
	if r.untilNext--; r.untilNext <= 0 {
		r.obs = append(r.obs, width)
		r.untilNext = 24 + (r.scroll*7)%15
	}

	if r.jump >= 0 {
		if r.jump++; r.jump >= jumpSpan {
			r.jump = -1
		}
	} else if cat != Distracting {
		// Only a focused ball sees the obstacle in time to jump.
		for _, o := range r.obs {
			if g := o - (ballX + 3); g >= 0 && g <= 3 {
				r.jump = 0
				break
			}
		}
	}
	if r.jump < 0 {
		for i, o := range r.obs {
			if o <= ballX+2 && o+obsW-1 >= ballX {
				r.bump = 8
				r.obs = append(r.obs[:i], r.obs[i+1:]...)
				break
			}
		}
	}
}

// Styles say what a cell is, so the renderer can colour it.
const (
	stNone uint8 = iota
	stThread
	stDash
	stBall
	stObstacle
	stScissors
	stBanner
	stSpark1
	stSpark2
	stSpark3
)

type canvas struct {
	r [][]rune
	s [][]uint8
}

func newCanvas(w int) canvas {
	c := canvas{r: make([][]rune, sceneRows), s: make([][]uint8, sceneRows)}
	for i := range c.r {
		c.r[i] = []rune(strings.Repeat(" ", w))
		c.s[i] = make([]uint8, w)
	}
	return c
}

// put writes text at (y, x), skipping spaces so sprites can overlap the thread.
func (c canvas) put(y, x int, text string, st uint8) {
	if y < 0 || y >= len(c.r) {
		return
	}
	for i, ch := range []rune(text) {
		if x+i >= 0 && x+i < len(c.r[y]) && ch != ' ' {
			c.r[y][x+i], c.s[y][x+i] = ch, st
		}
	}
}

// drawCanvas renders the world, w cells wide, with a style for every cell.
func (r *runner) drawCanvas(w int, idle bool) canvas {
	if w < minWidth {
		w = minWidth
	}
	c := newCanvas(w)

	if r.celeb >= 0 {
		for i := 0; i < 14; i++ {
			x := (i*7 + r.celeb*5 + i*i) % w
			y := (i*3 + r.celeb/2) % 3
			c.put(y, x, string("*+.'`"[(i+r.celeb)%5]), stSpark1+uint8(i%3))
		}
	}

	// The thread: solid behind the ball (already spun), dashed ahead (still to spin).
	for x := 0; x < w; x++ {
		switch {
		case x < ballX:
			c.r[groundRow][x], c.s[groundRow][x] = '~', stThread
		case (x+r.scroll)%4 < 2:
			c.r[groundRow][x], c.s[groundRow][x] = '-', stDash
		}
	}
	if r.cutF >= 20 {
		for x := cutX + 3; x < w; x++ {
			c.r[groundRow][x], c.s[groundRow][x] = ' ', stNone
		}
	}
	for _, o := range r.obs {
		c.put(ballRow, o, "[!]", stObstacle)
	}

	h := 0
	if r.jump >= 0 {
		h = jumpHeight[r.jump]
	}
	face := string("@&%&"[(r.scroll/2)%4])
	switch {
	case r.bump > 0:
		face = "x"
	case idle:
		face = "-"
	}
	c.put(ballRow-h, ballX, "("+face+")", stBall)

	if r.cutF >= 0 {
		if r.cutF >= 14 {
			c.put(0, 2, "G A M E   O V E R", stBanner)
		}
		var top int
		switch {
		case r.cutF < 12:
			top = -3 + r.cutF*5/11
		case r.cutF < 36:
			top = 2
		default:
			top = 2 - (r.cutF-36)*5/11
		}
		sprite := scissorsShut
		if r.cutF < 12 || r.cutF >= 36 || (r.cutF < 24 && (r.cutF/3)%2 == 0) {
			sprite = scissorsOpen
		}
		for i, line := range sprite {
			c.put(top+i, cutX-2, line, stScissors)
		}
	}
	return c
}

// draw renders the world as plain text rows.
func (r *runner) draw(w int, cat Category, idle bool) [sceneRows]string {
	c := r.drawCanvas(w, idle)
	var out [sceneRows]string
	for i := range c.r {
		out[i] = string(c.r[i])
	}
	return out
}

var banterLines = []string{
	"Atropos cut your %s thread after %s of thread. Not bad.",
	"%s used up its allotment. Atropos did the rest. %s of thread spun.",
	"Snip. %s is done for the day. %s of thread so far.",
}

func banter(app string, spun time.Duration) string {
	return fmt.Sprintf(banterLines[len(app)%len(banterLines)], app, liveDur(spun))
}
