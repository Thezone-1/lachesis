package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"
)

// demo plays a scripted loop of the live view: the wheel spins, the thread
// frays, Atropos cuts, and everything rests. It reads nothing from the machine.
func demo(args []string) error {
	if !isTerminal(os.Stdout) || !enableVT() {
		return fmt.Errorf("demo needs an interactive terminal")
	}
	opts := uiOpts{
		ascii: os.Getenv("LACHESIS_ASCII") != "",
		color: os.Getenv("NO_COLOR") == "",
		width: termWidth() - 8,
		pal:   paletteFor(loadConfig()),
	}
	if opts.width > 56 {
		opts.width = 56
	}

	type act struct {
		name   string
		frames int
		cat    Category
		idle   bool
		cut    bool
	}
	script := []act{
		{"running: you are on task", 110, Productive, false, false},
		{"jogging: a neutral app", 50, Neutral, false, false},
		{"stumbling: a distraction", 90, Distracting, false, false},
		{"Atropos arrives", cutFrames + 4, Productive, false, true},
		{"away: the yarn rests", 30, Neutral, true, false},
	}
	total := 0
	for _, a := range script {
		total += a.frames
	}

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	view := newLiveView(os.Stdout)
	defer view.close()

	ticker := time.NewTicker(125 * time.Millisecond)
	defer ticker.Stop()
	run := newRunner()
	frame := 0
	for {
		select {
		case <-stop:
			return nil
		case <-ticker.C:
			frame++
			t := frame % total
			var cur act
			local := 0
			for _, a := range script {
				if t < a.frames {
					cur, local = a, t
					break
				}
				t -= a.frames
			}
			if cur.cut && local == 0 {
				run.startCut("chrome")
			}
			if cur.name == script[0].name && local == 70 {
				run.startCelebrate()
			}
			run.step(frame, opts.width, cur.cat, cur.idle)
			l := live{
				sample:  Sample{App: "demo", Title: cur.name},
				cat:     cur.cat,
				idle:    cur.idle,
				m:       DayMetrics{Active: 3*time.Hour + 42*time.Minute, Idle: 25 * time.Minute},
				score:   Score{Focus: 0.72, Depth: 0.61, Cohesion: 0.88, Yield: 0.67, Total: 74},
				block:   80 * time.Minute,
				hi:      74,
				deep:    31 + frame/40,
				samples: frame/40 + 1,
				last:    time.Now().Add(-time.Duration(frame%40) * 125 * time.Millisecond),
				now:     time.Now(),
			}
			view.draw(renderLive(l, frame, run, opts))
		}
	}
}
