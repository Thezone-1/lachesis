package main

import (
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"
)

var sgrRe = regexp.MustCompile(`\x1b\[([0-9;]*)m`)

// allCodes renders every state of the UI and returns the SGR codes it used.
func allCodes(t *testing.T, o uiOpts) map[string]bool {
	t.Helper()
	codes := map[string]bool{}
	collect := func(lines []string) {
		for _, m := range sgrRe.FindAllStringSubmatch(strings.Join(lines, "\n"), -1) {
			codes[m[1]] = true
		}
	}
	for _, cat := range []Category{Productive, Neutral, Distracting} {
		for _, idle := range []bool{false, true} {
			r := newRunner()
			r.startCelebrate()
			simulate(r, 40, cat, idle)
			l := live{sample: Sample{App: "code", Title: "x"}, cat: cat, idle: idle, samples: 3,
				m: DayMetrics{Active: time.Hour}, score: Score{Total: 74}, cut: "chrome", warn: ""}
			collect(renderLive(l, 4, r, o))
			l.warn = "titles are off"
			collect(renderLive(l, 5, r, o))
		}
	}
	r := newRunner()
	r.startCut("chrome")
	simulate(r, 30, Productive, false)
	collect(renderLive(live{cat: Productive, m: DayMetrics{Active: time.Hour}, score: Score{Total: 50}}, 1, r, o))
	tempData(t)
	start := time.Now().Add(-time.Hour).Truncate(time.Second)
	writeSamples(t, start, 480, "code", "main.go")
	collect(sessionRecap(start, start.Add(41*time.Minute), "x", defaultConfig(), o))
	return codes
}

func TestDefaultThemeIsMonochrome(t *testing.T) {
	t.Setenv("LACHESIS_THEME", "")
	o := uiOpts{width: testWidth, color: true, pal: paletteFor(defaultConfig())}
	codes := allCodes(t, o)
	if len(codes) == 0 {
		t.Fatal("expected some styling")
	}
	grey := regexp.MustCompile(`^(?:0|1|(?:1;)?38;5;(\d+))$`)
	for c := range codes {
		m := grey.FindStringSubmatch(c)
		if m == nil {
			t.Fatalf("code %q is not a plain grey or bold", c)
		}
		if m[1] != "" {
			if n, _ := strconv.Atoi(m[1]); n < 232 && n != 255 && !(n >= 236 && n <= 255) {
				t.Fatalf("code %q is not on the grey ramp (232-255)", c)
			}
		}
	}
}

func TestColorThemeIsOptIn(t *testing.T) {
	t.Setenv("LACHESIS_THEME", "")
	if paletteFor(defaultConfig()) != &monoPalette {
		t.Fatal("the default must be mono")
	}
	cfg := defaultConfig()
	cfg.Theme = "color"
	if paletteFor(cfg) != &colorPalette {
		t.Fatal("config theme=color should select the colour palette")
	}
	t.Setenv("LACHESIS_THEME", "mono")
	if paletteFor(cfg) != &monoPalette {
		t.Fatal("the environment overrides the config")
	}
	t.Setenv("LACHESIS_THEME", "color")
	if paletteFor(defaultConfig()) != &colorPalette {
		t.Fatal("LACHESIS_THEME=color should select the colour palette")
	}
}

func TestColorThemeUsesHue(t *testing.T) {
	o := uiOpts{width: testWidth, color: true, pal: &colorPalette}
	codes := allCodes(t, o)
	hue := false
	for c := range codes {
		if strings.Contains(c, "196") || strings.Contains(c, "82") {
			hue = true
		}
	}
	if !hue {
		t.Fatal("the colour theme should still use colour")
	}
}
