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
		{"spinning: you are on task", 40, Productive, false, false},
		{"slowing: a neutral app", 32, Neutral, false, false},
		{"fraying: a distraction", 40, Distracting, false, false},
		{"Atropos arrives", cutFrames + 4, Productive, false, true},
		{"away: the wheel rests", 32, Neutral, true, false},
	}
	total := 0
	for _, a := range script {
		total += a.frames
	}

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	view := newLiveView(os.Stdout)
	defer view.close()

	start := time.Now()
	ticker := time.NewTicker(125 * time.Millisecond)
	defer ticker.Stop()
	pos, frame := 0, 0
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
			if threadStep(frame, cur.cat, cur.idle) {
				pos++
			}
			samples := frame/40 + 1
			l := live{
				sample:   Sample{App: "demo", Title: cur.name},
				cat:      cur.cat,
				idle:     cur.idle,
				m:        DayMetrics{Active: 3*time.Hour + 42*time.Minute, Idle: 25 * time.Minute},
				score:    Score{Focus: 0.72, Depth: 0.61, Cohesion: 0.88, Yield: 0.67, Total: 74},
				block:    80 * time.Minute,
				cutting:  cur.cut && local < cutFrames,
				cutFrame: local,
				cutApp:   "chrome",
				samples:  samples,
				last:     time.Now().Add(-time.Duration(frame%40) * 125 * time.Millisecond),
				now:      time.Now(),
			}
			_ = start
			view.draw(renderLive(l, frame, pos, opts))
		}
	}
}
