package main

import (
	"fmt"
	"os"
	"os/signal"
	"time"
)

const version = "0.2.0"

const usage = `lacheisis %s — measures each thread and decides how much is allotted

Usage:
  lacheisis track                    run the collector (foreground; use Start-Process to detach)
  lacheisis report [YYYY-MM-DD]      day's score, focus blocks, yield and per-app thread (default: today)
  lacheisis allot <app> <duration>   set a daily allotment, e.g. lacheisis allot chrome 2h
  lacheisis allot                    list allotments
  lacheisis classify <app|title> <pattern> <category>
                                     add a rule (category: productive|neutral|distracting)
  lacheisis classify                 list rules

Data: %s
`

func main() {
	if len(os.Args) < 2 {
		fmt.Printf(usage, version, baseDir())
		return
	}
	cfg := loadConfig()

	var err error
	switch os.Args[1] {
	case "track":
		err = track(cfg)
	case "report":
		day := time.Now().Format("2006-01-02")
		if len(os.Args) > 2 {
			day = os.Args[2]
		}
		err = printReport(day, cfg)
	case "allot":
		err = allot(cfg, os.Args[2:])
	case "classify":
		err = classify(cfg, os.Args[2:])
	case "version", "--version":
		fmt.Println("lacheisis", version)
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
func track(cfg Config) error {
	fmt.Printf("lacheisis %s — spinning the thread (every %ds, idle after %ds)\n",
		version, cfg.PollSeconds, cfg.IdleSeconds)
	fmt.Println("data:", dataDir())

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt)
	ticker := time.NewTicker(time.Duration(cfg.PollSeconds) * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-stop:
			fmt.Println("thread paused")
			return nil
		case now := <-ticker.C:
			s := Sample{TS: now}
			if IdleDuration() >= time.Duration(cfg.IdleSeconds)*time.Second {
				s.Idle = true
			} else if app, title, err := ActiveWindow(); err == nil {
				s.App, s.Title = app, title
			} else {
				s.Idle = true // locked screen / secure desktop counts as away
			}
			if err := appendSample(s); err != nil {
				fmt.Fprintln(os.Stderr, "write:", err)
			}
		}
	}
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
		return fmt.Errorf("usage: lacheisis allot <app> <duration>")
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
