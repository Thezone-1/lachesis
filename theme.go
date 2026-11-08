package main

import (
	"os"
	"strings"
)

// palette holds every colour the UI uses, as SGR codes, so a theme is one value.
type palette struct {
	bold, dim, label, hint, box                            string
	hi, today, beat                                        string
	thread, dash, obstacle, scissors, banner               string
	spark                                                  [3]string
	ballProductive, ballNeutral, ballDistracting, ballIdle string
	title                                                  []int // gradient across the title letters
	bar                                                    [10]int
	barOff                                                 string
	warn, cut, celebrate, ok, bad                          string
}

// monoPalette is the default: greys and white, with meaning carried by brightness
// (bright is focused, dim is distracted) rather than by hue.
var monoPalette = palette{
	bold: "1", dim: "38;5;240", label: "38;5;244", hint: "38;5;245", box: "38;5;245",
	hi: "38;5;245", today: "1;38;5;255", beat: "1;38;5;255",
	thread: "38;5;252", dash: "38;5;238", obstacle: "38;5;245", scissors: "1;38;5;255", banner: "1;38;5;255",
	spark:          [3]string{"1;38;5;255", "38;5;250", "38;5;245"},
	ballProductive: "1;38;5;255", ballNeutral: "38;5;249", ballDistracting: "38;5;243", ballIdle: "38;5;240",
	title:  []int{243, 245, 247, 249, 251, 253, 255, 255},
	bar:    [10]int{238, 240, 242, 244, 246, 248, 250, 252, 254, 255},
	barOff: "38;5;236",
	warn:   "38;5;250", cut: "1;38;5;255", celebrate: "1;38;5;255", ok: "1;38;5;255", bad: "38;5;245",
}

// colorPalette is the opt-in colourful theme.
var colorPalette = palette{
	bold: "1", dim: "38;5;240", label: "38;5;244", hint: "38;5;214", box: "38;5;220",
	hi: "38;5;220", today: "1;38;5;231", beat: "1;38;5;82",
	thread: "38;5;220", dash: "38;5;240", obstacle: "1;38;5;196", scissors: "1;38;5;231", banner: "1;38;5;196",
	spark:          [3]string{"1;38;5;227", "1;38;5;213", "1;38;5;51"},
	ballProductive: "1;38;5;82", ballNeutral: "1;38;5;214", ballDistracting: "1;38;5;203", ballIdle: "38;5;244",
	title:  []int{213, 177, 141, 105, 69, 33, 39, 45},
	bar:    [10]int{196, 202, 208, 214, 220, 226, 190, 154, 118, 82},
	barOff: "38;5;238",
	warn:   "33", cut: "31", celebrate: "1;38;5;227", ok: "1;38;5;82", bad: "1;38;5;203",
}

// paletteFor picks the theme: LACHESIS_THEME overrides the config, and mono is the default.
func paletteFor(cfg Config) *palette {
	name := strings.ToLower(os.Getenv("LACHESIS_THEME"))
	if name == "" {
		name = strings.ToLower(cfg.Theme)
	}
	if name == "color" || name == "colour" {
		return &colorPalette
	}
	return &monoPalette
}

// p returns the options' palette, falling back to mono.
func (o uiOpts) p() palette {
	if o.pal == nil {
		return monoPalette
	}
	return *o.pal
}

// termOpts are the display options for a one-shot command writing to the terminal.
func termOpts(cfg Config) uiOpts {
	return uiOpts{
		ascii: os.Getenv("LACHESIS_ASCII") != "",
		color: isTerminal(os.Stdout) && os.Getenv("NO_COLOR") == "",
		pal:   paletteFor(cfg),
	}
}
