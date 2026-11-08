package main

import (
	"strings"
	"testing"
	"time"
)

func TestThreadStep(t *testing.T) {
	if threadStep(1, Productive, true) {
		t.Fatal("idle must freeze the thread")
	}
	if !threadStep(1, Productive, false) {
		t.Fatal("productive thread advances every frame")
	}
	if threadStep(1, Neutral, false) || !threadStep(3, Neutral, false) {
		t.Fatal("neutral thread advances every third frame")
	}
}

func TestThreadLineWidthAndCut(t *testing.T) {
	for _, cat := range []Category{Productive, Neutral, Distracting} {
		for _, cut := range []bool{false, true} {
			got := []rune(threadLine(40, 7, cat, false, cut, false))
			if len(got) != 40 {
				t.Fatalf("%s cut=%v width %d", cat, cut, len(got))
			}
			if cut && !strings.ContainsRune(string(got), '✂') {
				t.Fatal("cut thread must show scissors")
			}
		}
	}
}

func TestRenderLiveASCIIHasNoWideGlyphs(t *testing.T) {
	l := live{sample: Sample{App: "code", Title: "main.go"}, cat: Productive,
		score: Score{Total: 74}, block: 80 * time.Minute, cut: "slack"}
	lines := renderLive(l, 5, 3, uiOpts{ascii: true, width: 50})
	for _, line := range lines {
		for _, r := range line {
			if r > 127 && r != '·' && r != '✂' && r != '…' {
				t.Fatalf("non-ASCII glyph %q in ascii mode: %s", r, line)
			}
		}
	}
	if !strings.Contains(strings.Join(lines, "\n"), "main.go") {
		t.Fatal("current window title should be shown")
	}
}

func TestRenderLiveZeroStateDoesNotPanic(t *testing.T) {
	lines := renderLive(live{idle: true}, 0, 0, uiOpts{width: 10})
	if len(lines) == 0 {
		t.Fatal("expected a frame")
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

func TestHideTitlesAndWarmup(t *testing.T) {
	l := live{sample: Sample{App: "chrome", Title: "secret tab"}, cat: Neutral}
	all := strings.Join(renderLive(l, 1, 1, uiOpts{width: 50, hideTitles: true}), "\n")
	if strings.Contains(all, "secret tab") {
		t.Fatal("window title leaked with hideTitles")
	}
	if !strings.Contains(all, "warming up") {
		t.Fatal("score must say warming up before enough activity")
	}
	l.m.Active = 10 * time.Minute
	l.score.Total = 60
	if strings.Contains(strings.Join(renderLive(l, 1, 1, uiOpts{width: 50}), "\n"), "warming up") {
		t.Fatal("score should show once there is enough activity")
	}
}

func TestCutAnimationStages(t *testing.T) {
	const w = 40
	start := w - scW
	rest := buildScene(w, 0, Productive, false, -1, false, true)
	if !strings.Contains(strings.Join(rest.thread[:], "\n"), "X") {
		t.Fatal("Atropos must always hold visible scissors")
	}
	shut := buildScene(w, 0, Productive, false, -1, false, false)
	if shut.thread == rest.thread {
		t.Fatal("scissors should snip between open and shut")
	}
	walking := buildScene(w, 0, Productive, false, 6, false, true)
	if strings.Index(walking.thread[threadRow], "X") >= start {
		t.Fatal("scissors should have left their resting place")
	}
	cut := buildScene(w, 0, Productive, false, 24, false, true)
	mid := start / 2
	if strings.TrimSpace(cut.thread[threadRow][mid+scW:]) != "" {
		t.Fatal("thread right of the cut should be gone")
	}
	if !strings.Contains(strings.Join(cut.thread[threadRow+1:], ""), "~~") {
		t.Fatal("the severed tail should be drawn below the thread")
	}
	for f := 0; f < cutFrames; f++ {
		s := buildScene(w, 3, Neutral, false, f, true, f%2 == 0)
		for _, row := range s.thread {
			if len([]rune(row)) != w {
				t.Fatalf("frame %d: row width %d", f, len([]rune(row)))
			}
		}
	}
}

func TestYarnSpins(t *testing.T) {
	seen := map[string]bool{}
	for p := 0; p < 4; p++ {
		f := yarnFrame(p)
		for _, row := range f {
			if len([]rune(row)) != yarnW {
				t.Fatalf("phase %d: yarn row width %d", p, len([]rune(row)))
			}
		}
		seen[strings.Join(f[:], "\n")] = true
	}
	if len(seen) != 4 {
		t.Fatalf("yarn should look different in each of 4 phases, got %d", len(seen))
	}
	first := yarnFrame(0)
	if !strings.Contains(strings.Join(first[:], ""), "~~") {
		t.Fatal("a strand should lead from the yarn to the wheel")
	}
}
