package main

import (
	"strings"
	"testing"
	"time"
)

func writeSamples(t *testing.T, start time.Time, n int, app, title string) {
	t.Helper()
	for i := 0; i < n; i++ {
		if err := appendSample(Sample{TS: start.Add(time.Duration(i) * 5 * time.Second), App: app, Title: title}); err != nil {
			t.Fatal(err)
		}
	}
}

func TestRecapScoresAFullSession(t *testing.T) {
	tempData(t)
	start := time.Now().Add(-time.Hour).Truncate(time.Second)
	writeSamples(t, start, 480, "code", "main.go") // 40 minutes of focus
	end := start.Add(41 * time.Minute)

	cfg := defaultConfig()
	out := strings.Join(sessionRecap(start, end, "deep work", cfg, uiOpts{ascii: true}), "\n")
	for _, want := range []string{"session complete", "deep work", "on the clock", "deep block", "score"} {
		if !strings.Contains(out, want) {
			t.Fatalf("recap is missing %q:\n%s", want, out)
		}
	}
	if strings.Contains(out, "too short") {
		t.Fatalf("40 minutes should be scored:\n%s", out)
	}
}

func TestRecapOfAShortSessionSaysSo(t *testing.T) {
	tempData(t)
	start := time.Now().Add(-5 * time.Minute).Truncate(time.Second)
	writeSamples(t, start, 6, "code", "main.go")
	out := strings.Join(sessionRecap(start, start.Add(time.Minute), "", defaultConfig(), uiOpts{ascii: true}), "\n")
	if !strings.Contains(out, "too short to score") {
		t.Fatalf("a short session must not get a score:\n%s", out)
	}
}

func TestRecapCardIsRectangular(t *testing.T) {
	tempData(t)
	start := time.Now().Add(-time.Hour).Truncate(time.Second)
	writeSamples(t, start, 480, "code", "main.go")
	lines := sessionRecap(start, start.Add(41*time.Minute), "x", defaultConfig(), uiOpts{ascii: true})
	want := len([]rune(lines[0]))
	for _, l := range lines {
		if len([]rune(l)) != want {
			t.Fatalf("ragged card: %d vs %d\n%s", len([]rune(l)), want, strings.Join(lines, "\n"))
		}
	}
}

func TestRecapWithNoDataDoesNotPanic(t *testing.T) {
	tempData(t)
	now := time.Now()
	if len(sessionRecap(now, now.Add(time.Minute), "", defaultConfig(), uiOpts{})) == 0 {
		t.Fatal("expected a card")
	}
}
