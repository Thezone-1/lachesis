package main

import (
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"
)

var version = "0.3.0-dev"

const usage = `lachesis %s — measures each thread and decides how much is allotted

Usage:
  lachesis track [--plain] [--hide-titles]
                                     run the collector with a live view (--plain: quiet output;
                                     --hide-titles: for screen recordings)
  lachesis report [YYYY-MM-DD]      day's score, focus blocks, yield and per-app thread (default: today)
  lachesis demo                      play the story of the thread (no tracking, no permissions)
  lachesis allot <app> <duration>   set a daily allotment, e.g. lachesis allot chrome 2h
  lachesis allot                    list allotments
  lachesis classify <app|title> <pattern> <category>
                                     add a rule (category: productive|neutral|distracting)
  lachesis classify                 list rules

Data: %s
`

func main() {
	if len(os.Args) < 2 {
		fmt.Printf(usage, version, baseDir())
		return
	}
	migrateLegacyDir()
	cfg := loadConfig()

	var err error
	switch os.Args[1] {
	case "track":
		err = track(cfg, os.Args[2:])
	case "report":
		day := time.Now().Format("2006-01-02")
		if len(os.Args) > 2 {
			day = os.Args[2]
		}
		err = printReport(day, cfg)
	case "demo":
		err = demo(os.Args[2:])
	case "allot":
		err = allot(cfg, os.Args[2:])
	case "classify":
		err = classify(cfg, os.Args[2:])
	case "version", "--version":
		fmt.Println("lachesis", version)
	default:
		fmt.Printf(usage, version, baseDir())
		os.Exit(2)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

// track samples the foreground window every poll interval until interrupted.
// On a terminal it draws a live view; otherwise (or with --plain) it prints a
// single status line and stays quiet.
func track(cfg Config, args []string) error {
	if err := preflight(); err != nil {
		return err
	}

	animate := isTerminal(os.Stdout) && os.Getenv("LACHESIS_PLAIN") == ""
	hideTitles := false
	for _, a := range args {
		switch a {
		case "--plain":
			animate = false
		case "--hide-titles":
			hideTitles = true
		}
	}
	if animate && !enableVT() {
		animate = false
	}

	today := time.Now().Format("2006-01-02")
	samples, _ := readDay(today)

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	if !animate {
		fmt.Printf("lachesis %s: spinning the thread (every %ds, idle after %ds)\n",
			version, cfg.PollSeconds, cfg.IdleSeconds)
		fmt.Println("data:", dataDir())
	}

	opts := uiOpts{
		ascii:      os.Getenv("LACHESIS_ASCII") != "",
		color:      os.Getenv("NO_COLOR") == "",
		width:      termWidth() - 8,
		hideTitles: hideTitles,
	}
	if opts.width > 56 {
		opts.width = 56
	}
	var view *liveView
	if animate {
		view = newLiveView(os.Stdout)
		defer view.close()
	}

	state := live{}
	frame := 0
	run := newRunner()
	hi := 0
	if animate {
		hi = bestDeepMinutes(cfg, today)
	}
	fired := map[string]bool{}
	var lastErr string
	var lastTS time.Time
	sampleNow := func(now time.Time) {
		if d := now.Format("2006-01-02"); d != today {
			today, samples, fired = d, nil, map[string]bool{} // midnight: start a fresh day
		}
		s := Sample{TS: now}
		if IdleDuration() >= time.Duration(cfg.IdleSeconds)*time.Second {
			s.Idle = true
		} else if app, title, err := ActiveWindow(); err == nil {
			s.App, s.Title = app, title
			lastErr = ""
		} else {
			s.Idle = true // locked screen / secure desktop counts as away
			if err != errLocked {
				lastErr = "cannot read the front window (check your terminal's Automation/Accessibility permission)"
			}
		}
		if err := appendSample(s); err != nil {
			fmt.Fprintln(os.Stderr, "write:", err)
		}
		samples = append(samples, s)
		lastTS = now
		if animate {
			state = buildLive(samples, s, cfg)
			state.hi = hi
			if state.cut != "" && !fired[state.cut] {
				fired[state.cut] = true
				run.startCut(state.cut)
			}
		}
	}
	sampleNow(time.Now())

	poll := time.NewTicker(time.Duration(cfg.PollSeconds) * time.Second)
	defer poll.Stop()
	var frames <-chan time.Time
	if animate {
		ft := time.NewTicker(125 * time.Millisecond)
		defer ft.Stop()
		frames = ft.C
	}

	for {
		select {
		case <-stop:
			if !animate {
				fmt.Println("thread paused")
			}
			return nil
		case now := <-poll.C:
			sampleNow(now)
		case <-frames:
			frame++
			run.step(frame, opts.width, state.cat, state.idle)
			shown := state
			shown.samples, shown.last, shown.now, shown.warn = len(samples), lastTS, time.Now(), lastErr
			view.draw(renderLive(shown, frame, run, opts))
		}
	}
}

// buildLive derives the view state from today's samples.
func buildLive(samples []Sample, last Sample, cfg Config) live {
	m := sessionize(samples, cfg.PollSeconds, cfg.Rules)
	l := live{sample: last, idle: last.Idle || last.App == "", m: m, score: computeScore(m, cfg.Weights),
		deep: int(m.DeepTotal.Minutes())}
	if !l.idle {
		l.cat = classifySample(last, cfg.Rules)
	}
	if n := len(m.Blocks); n > 0 {
		l.block = m.Blocks[n-1].Dur
	}
	stats, _, _ := buildReport(samples, cfg.PollSeconds, cfg.Rules)
	for _, st := range stats {
		if a := cfg.allotmentFor(st.app); a > 0 && st.dur > a {
			l.cut = st.app
			break
		}
	}
	return l
}

func termWidth() int {
	if c, err := strconv.Atoi(os.Getenv("COLUMNS")); err == nil && c > 0 {
		return c
	}
	return 64
}

func allot(cfg Config, args []string) error {
	if len(args) == 0 {
		if len(cfg.Allotments) == 0 {
			fmt.Println("no allotments set")
			return nil
		}
		for app, d := range cfg.Allotments {
			fmt.Printf("  %-22s %s/day\n", app, d)
		}
		return nil
	}
	if len(args) != 2 {
		return fmt.Errorf("usage: lachesis allot <app> <duration>")
	}
	app, durStr := args[0], args[1]
	if _, err := time.ParseDuration(durStr); err != nil {
		return fmt.Errorf("bad duration %q: %w", durStr, err)
	}
	cfg.Allotments[app] = durStr
	if err := saveConfig(cfg); err != nil {
		return err
	}
	fmt.Printf("Lachesis allots %s to %s each day\n", durStr, app)
	return nil
}
