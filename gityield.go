package main

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// Commit is one artifact found in a configured repo on the report day.
type Commit struct {
	Repo    string // repo basename
	Hash    string
	TS      time.Time
	Subject string
	Files   int // files changed, from --shortstat
}

// commitAttachWindow is how long after a deep-work block ends a commit is
// still attributed to it (you commit shortly after finishing the work).
const commitAttachWindow = 30 * time.Minute

// dayCommits scans the configured repos (read-only) for commits made on the
// given local day. Repos that fail to scan are skipped silently — a moved
// or offline repo must not break the report.
func dayCommits(repos []string, day string) []Commit {
	var commits []Commit
	for _, repo := range repos {
		commits = append(commits, repoCommits(repo, day)...)
	}
	return commits
}

func repoCommits(repo, day string) []Commit {
	out, err := exec.Command("git", "-C", repo, "log",
		"--since", day+" 00:00:00", "--until", day+" 23:59:59",
		"--pretty=format:%H|%ct|%s", "--shortstat").Output()
	if err != nil {
		return nil
	}
	var commits []Commit
	name := filepath.Base(repo)
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if h, ct, subj, ok := parseCommitLine(line); ok {
			commits = append(commits, Commit{Repo: name, Hash: h, TS: ct, Subject: subj, Files: 0})
			continue
		}
		// shortstat line for the preceding commit: "3 files changed, ..."
		if len(commits) > 0 && strings.Contains(line, "changed") {
			commits[len(commits)-1].Files = parseFilesChanged(line)
		}
	}
	return commits
}

func parseCommitLine(line string) (hash string, ts time.Time, subject string, ok bool) {
	parts := strings.SplitN(line, "|", 3)
	if len(parts) != 3 || len(parts[0]) != 40 {
		return "", time.Time{}, "", false
	}
	unix, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		return "", time.Time{}, "", false
	}
	return parts[0], time.Unix(unix, 0), parts[2], true
}

func parseFilesChanged(line string) int {
	// " 3 files changed, 40 insertions(+)" — first field is the count
	fields := strings.Fields(line)
	if len(fields) == 0 {
		return 0
	}
	n, _ := strconv.Atoi(fields[0])
	return n
}

// attributeCommits attaches each commit to the deep-work block containing
// its timestamp, or the nearest block that ended at most 30 min before it.
// Returns the commits that found no block (yield the report still lists).
func attributeCommits(blocks []Block, commits []Commit) (orphans []Commit) {
	for _, c := range commits {
		best := -1
		var bestGap time.Duration
		for i := range blocks {
			b := &blocks[i]
			if !b.Deep() {
				continue
			}
			if !c.TS.Before(b.Start) && c.TS.Before(b.End) {
				best = i
				bestGap = 0
				break
			}
			gap := c.TS.Sub(b.End)
			if gap > 0 && gap <= commitAttachWindow && (best < 0 || gap < bestGap) {
				best = i
				bestGap = gap
			}
		}
		if best >= 0 {
			blocks[best].Commits = append(blocks[best].Commits, c)
		} else {
			orphans = append(orphans, c)
		}
	}
	return orphans
}

// yieldRatio is the score component: artifact-linked deep-work time over
// total deep-work time. Ratio, never volume.
func yieldRatio(blocks []Block) float64 {
	var deep, linked time.Duration
	for _, b := range blocks {
		if !b.Deep() {
			continue
		}
		deep += b.Dur
		if len(b.Commits) > 0 {
			linked += b.Dur
		}
	}
	if deep == 0 {
		return 0
	}
	r := linked.Seconds() / deep.Seconds()
	if r > 1 {
		r = 1
	}
	return r
}

// yieldLine summarizes commits + files changed per deep-work hour for the
// report header (display only, never scored — volume gets farmed).
func yieldLine(blocks []Block, deepTotal time.Duration) string {
	var n, files int
	for _, b := range blocks {
		for _, c := range b.Commits {
			n++
			files += c.Files
		}
	}
	if n == 0 || deepTotal == 0 {
		return ""
	}
	h := deepTotal.Hours()
	return fmt.Sprintf("%d commits · %d files touched · %.1f commits per deep hour", n, files, float64(n)/h)
}
