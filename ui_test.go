package main

import (
	"strings"
	"testing"
	"time"
)

const testWidth = 56

func simulate(r *runner, frames int, cat Category, idle bool) (bumps int) {
	for f := 1; f <= frames; f++ {
		before := r.bump
		r.step(f, testWidth, cat, idle)
		if r.bump > before {
			bumps++
		}
	}
	return bumps
}

func TestAwayFreezesTheWorld(t *testing.T) {
	r := newRunner()
	simulate(r, 200, Productive, false)
	scroll := r.scroll
	simulate(r, 200, Productive, true)
	if r.scroll != scroll {
		t.Fatal("the world must not move while you are away")
	}
}

func TestFocusedBallClearsEveryObstacle(t *testing.T) {
	r := newRunner()
	if n := simulate(r, 3000, Productive, false); n != 0 {
		t.Fatalf("a productive ball bumped %d times", n)
	}
	if r.scroll < 2000 {
		t.Fatal("productive runs should keep moving")
	}
}

func TestNeutralBallClearsObstaclesToo(t *testing.T) {
	r := newRunner()
	if n := simulate(r, 3000, Neutral, false); n != 0 {
		t.Fatalf("a neutral ball bumped %d times", n)
	}
}

func TestDistractedBallStumbles(t *testing.T) {
	r := newRunner()
	if n := simulate(r, 3000, Distracting, false); n == 0 {
		t.Fatal("a distracted ball should hit obstacles")
	}
}

func TestScissorsScene(t *testing.T) {
	r := newRunner()
	r.startCut("chrome")
	for f := 1; f < 30; f++ {
		r.step(f, testWidth, Productive, false)
	}
	scene := r.draw(testWidth, Productive, false)
	all := strings.Join(scene[:], "\n")
	if !strings.Contains(all, "G A M E   O V E R") {
		t.Fatal("game over banner should show once the snipping starts")
	}
	if !strings.Contains(all, " o o ") || !strings.Contains(all, "|") {
		t.Fatal("scissors should be on screen")
	}
	if strings.TrimSpace(scene[groundRow][cutX+3:]) != "" {
		t.Fatal("the thread beyond the cut should be gone")
	}
	for f := 30; f <= cutFrames+2; f++ {
		r.step(f, testWidth, Productive, false)
	}
	if r.cutting() || len(r.obs) != 0 {
		t.Fatal("the scene should reset after the cut")
	}
	after := r.draw(testWidth, Productive, false)
	if strings.Contains(strings.Join(after[:], ""), "G A M E") {
		t.Fatal("the banner should be gone after the reset")
	}
}

func TestRowsAreAlwaysTheRightWidth(t *testing.T) {
	r := newRunner()
	r.startCut("x")
	for f := 1; f < 600; f++ {
		cat := []Category{Productive, Neutral, Distracting}[(f/40)%3]
		r.step(f, testWidth, cat, f%97 == 0)
		for _, row := range r.draw(testWidth, cat, false) {
			if n := len([]rune(row)); n != testWidth {
				t.Fatalf("frame %d: row is %d wide, want %d", f, n, testWidth)
			}
		}
	}
}

func TestRenderLiveShowsHeartbeatAndHidesTitles(t *testing.T) {
	l := live{sample: Sample{App: "chrome", Title: "secret tab"}, cat: Neutral, samples: 1,
		last: time.Now(), now: time.Now(), hi: 74, deep: 31}
	lines := renderLive(l, 1, newRunner(), uiOpts{width: testWidth, hideTitles: true, ascii: true})
	all := strings.Join(lines, "\n")
	if strings.Contains(all, "secret tab") {
		t.Fatal("window title leaked with hideTitles")
	}
	for _, want := range []string{"HI 00074", "00031", "tracking", "1 sample", "warming up"} {
		if !strings.Contains(all, want) {
			t.Fatalf("missing %q in:\n%s", want, all)
		}
	}
}

func TestWarningReplacesOtherMessages(t *testing.T) {
	l := live{warn: "cannot read the front window", cut: "chrome"}
	all := strings.Join(renderLive(l, 1, newRunner(), uiOpts{width: testWidth}), "\n")
	if !strings.Contains(all, "cannot read the front window") || strings.Contains(all, "over its allotment") {
		t.Fatal("a permission warning must take priority")
	}
}

func TestScoreBar(t *testing.T) {
	if got := scoreBar(74, true); got != "#######---" {
		t.Fatalf("got %q", got)
	}
	if got := scoreBar(150, true); got != "##########" {
		t.Fatalf("overflow must clamp, got %q", got)
	}
}

func TestBanterNamesTheApp(t *testing.T) {
	for _, app := range []string{"a", "bb", "ccc"} {
		if !strings.Contains(banter(app, 31*time.Minute), app) {
			t.Fatalf("banter should name %q", app)
		}
	}
}

func TestCelebrationSparklesAndResets(t *testing.T) {
	r := newRunner()
	r.startCelebrate()
	for f := 1; f < celebFrames/2; f++ {
		r.step(f, testWidth, Productive, false)
	}
	c := r.drawCanvas(testWidth, false)
	found := false
	for y := range c.s {
		for _, st := range c.s[y] {
			if st >= stSpark1 {
				found = true
			}
		}
	}
	if !found {
		t.Fatal("sparkles should be drawn while celebrating")
	}
	for f := celebFrames / 2; f < celebFrames+2; f++ {
		r.step(f, testWidth, Productive, false)
	}
	if r.celebrating() {
		t.Fatal("the celebration should end")
	}
}

func TestColoursOnlyWhenEnabled(t *testing.T) {
	l := live{sample: Sample{App: "code"}, cat: Productive, m: DayMetrics{Active: time.Hour}, score: Score{Total: 74}}
	plain := strings.Join(renderLive(l, 1, newRunner(), uiOpts{width: testWidth}), "\n")
	if strings.Contains(plain, "\x1b[") {
		t.Fatal("no escape codes when colour is off")
	}
	coloured := strings.Join(renderLive(l, 1, newRunner(), uiOpts{width: testWidth, color: true}), "\n")
	if !strings.Contains(coloured, "\x1b[1;38;5;") {
		t.Fatal("expected 256-colour codes when colour is on")
	}
}
