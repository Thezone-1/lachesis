package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

// A session is one run of the tracker. It is recorded in a pid file while it
// is running and in an append-only log, so start, stop and status work the
// same for a foreground `track` and a background `start`.

type sessionInfo struct {
	PID   int       `json:"pid"`
	Start time.Time `json:"start"`
	Label string    `json:"label,omitempty"`
}

type sessionEvent struct {
	Event string    `json:"event"` // "start" or "stop"
	TS    time.Time `json:"ts"`
	PID   int       `json:"pid,omitempty"`
	Label string    `json:"label,omitempty"`
}

func pidPath() string      { return filepath.Join(baseDir(), "lachesis.pid") }
func stopPath() string     { return filepath.Join(baseDir(), "stop.req") }
func sessionsPath() string { return filepath.Join(baseDir(), "sessions.jsonl") }
func logPath() string      { return filepath.Join(baseDir(), "lachesis.log") }

func readSession() (sessionInfo, bool) {
	data, err := os.ReadFile(pidPath())
	if err != nil {
		return sessionInfo{}, false
	}
	var info sessionInfo
	if json.Unmarshal(data, &info) != nil || info.PID <= 0 {
		return sessionInfo{}, false
	}
	return info, true
}

// runningSession returns the live session, clearing a stale pid file left by a crash.
func runningSession() (sessionInfo, bool) {
	info, ok := readSession()
	if !ok {
		return sessionInfo{}, false
	}
	if processAlive(info.PID) {
		return info, true
	}
	_ = os.Remove(pidPath())
	return sessionInfo{}, false
}

func logEvent(ev sessionEvent) {
	f, err := os.OpenFile(sessionsPath(), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return
	}
	defer f.Close()
	if line, err := json.Marshal(ev); err == nil {
		f.Write(append(line, '\n'))
	}
}

// acquireSession registers this process as the one running session.
func acquireSession(label string) (sessionInfo, error) {
	if err := os.MkdirAll(baseDir(), 0o700); err != nil {
		return sessionInfo{}, err
	}
	if cur, ok := runningSession(); ok && cur.PID != os.Getpid() {
		return sessionInfo{}, fmt.Errorf("already tracking since %s (pid %d); run `lachesis stop` first",
			cur.Start.Format("15:04"), cur.PID)
	}
	info := sessionInfo{PID: os.Getpid(), Start: time.Now(), Label: label}
	data, _ := json.Marshal(info)
	tmp := pidPath() + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return sessionInfo{}, err
	}
	if err := os.Rename(tmp, pidPath()); err != nil {
		return sessionInfo{}, err
	}
	_ = os.Remove(stopPath())
	logEvent(sessionEvent{Event: "start", TS: info.Start, PID: info.PID, Label: label})
	return info, nil
}

func releaseSession(info sessionInfo) {
	if cur, ok := readSession(); ok && cur.PID == info.PID {
		_ = os.Remove(pidPath())
	}
	_ = os.Remove(stopPath())
	logEvent(sessionEvent{Event: "stop", TS: time.Now(), PID: info.PID, Label: info.Label})
}

// stopRequested reports whether `lachesis stop` has asked the tracker to finish.
func stopRequested() bool {
	_, err := os.Stat(stopPath())
	return err == nil
}

// lastSession finds the most recent completed session in the log.
func lastSession() (start, end time.Time, label string, ok bool) {
	data, err := os.ReadFile(sessionsPath())
	if err != nil {
		return
	}
	var evs []sessionEvent
	for _, line := range splitLines(data) {
		var ev sessionEvent
		if json.Unmarshal(line, &ev) == nil {
			evs = append(evs, ev)
		}
	}
	for i := len(evs) - 1; i >= 0; i-- {
		if evs[i].Event != "stop" {
			continue
		}
		for j := i - 1; j >= 0; j-- {
			if evs[j].Event == "start" && evs[j].PID == evs[i].PID {
				return evs[j].TS, evs[i].TS, evs[j].Label, true
			}
		}
	}
	return
}

func splitLines(b []byte) [][]byte {
	var out [][]byte
	start := 0
	for i, c := range b {
		if c == '\n' {
			if i > start {
				out = append(out, b[start:i])
			}
			start = i + 1
		}
	}
	if start < len(b) {
		out = append(out, b[start:])
	}
	return out
}

// startCmd launches `lachesis track` detached from the terminal.
func startCmd(args []string) error {
	label := flagValue(args, "--label")
	if cur, ok := runningSession(); ok {
		fmt.Printf("already tracking since %s (pid %d)\n", cur.Start.Format("15:04"), cur.PID)
		return nil
	}
	if err := preflight(); err != nil {
		return err
	}
	if err := os.MkdirAll(baseDir(), 0o700); err != nil {
		return err
	}
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	logf, err := os.OpenFile(logPath(), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	defer logf.Close()

	argv := []string{"track", "--plain"}
	if label != "" {
		argv = append(argv, "--label", label)
	}
	cmd := exec.Command(exe, argv...)
	cmd.Stdout, cmd.Stderr = logf, logf
	detach(cmd)
	if err := cmd.Start(); err != nil {
		return err
	}
	pid := cmd.Process.Pid
	_ = cmd.Process.Release()

	for i := 0; i < 40; i++ {
		if cur, ok := runningSession(); ok && cur.PID == pid {
			o := uiOpts{color: isTerminal(os.Stdout) && os.Getenv("NO_COLOR") == ""}
			fmt.Printf("%s Lachesis is spinning the thread (pid %d). End the session with `lachesis stop`.\n", paint(o, "1;38;5;82", "●"), pid)
			if note := titleNote(); note != "" {
				fmt.Println("  note: " + note)
			}
			return nil
		}
		time.Sleep(100 * time.Millisecond)
	}
	return fmt.Errorf("the tracker did not start; see %s", logPath())
}

// stopCmd asks the running session to finish and waits for it.
func stopCmd(cfg Config) error {
	cur, ok := runningSession()
	if !ok {
		fmt.Println("not tracking")
		return nil
	}
	if err := os.WriteFile(stopPath(), nil, 0o600); err != nil {
		return err
	}
	deadline := time.Now().Add(time.Duration(cfg.PollSeconds+3) * time.Second)
	for time.Now().Before(deadline) {
		if _, still := runningSession(); !still {
			fmt.Printf("thread paused after %s\n", liveDur(time.Since(cur.Start)))
			printRecap(cur.Start, time.Now(), cur.Label, cfg)
			return nil
		}
		time.Sleep(100 * time.Millisecond)
	}
	if p, err := os.FindProcess(cur.PID); err == nil {
		_ = p.Kill()
	}
	_ = os.Remove(pidPath())
	_ = os.Remove(stopPath())
	fmt.Printf("the tracker did not answer, so it was stopped by force (pid %d)\n", cur.PID)
	return nil
}

func statusCmd() error {
	if cur, ok := runningSession(); ok {
		label := ""
		if cur.Label != "" {
			label = fmt.Sprintf(" · %q", cur.Label)
		}
		fmt.Printf("tracking since %s (%s) · pid %d%s\n", cur.Start.Format("15:04"), liveDur(time.Since(cur.Start)), cur.PID, label)
		return nil
	}
	fmt.Println("not tracking")
	if s, e, label, ok := lastSession(); ok {
		l := ""
		if label != "" {
			l = fmt.Sprintf(" · %q", label)
		}
		fmt.Printf("last session %s to %s (%s)%s\n", s.Format("Jan 2 15:04"), e.Format("15:04"), liveDur(e.Sub(s)), l)
	}
	return nil
}

// flagValue returns the value after name in args, or "".
func flagValue(args []string, name string) string {
	for i, a := range args {
		if a == name && i+1 < len(args) {
			return args[i+1]
		}
	}
	return ""
}
