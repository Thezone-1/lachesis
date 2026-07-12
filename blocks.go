package main

import (
	"time"
)

const deepWorkMin = 25 * time.Minute

// Block is a contiguous run of attention on one app. Foreign interruptions
// of at most glanceSamples samples (alt-tab glances) are absorbed; idle
// samples and recording gaps always break blocks.
type Block struct {
	App      string
	Category Category
	Start    time.Time
	End      time.Time
	Dur      time.Duration
	Samples  int            // same-app samples only
	Titles   map[string]int // title -> sample count, entropy input
	Commits  []Commit       // attributed by gityield
}

const glanceSamples = 2

// Entropy is unique titles over samples: a farmer looping one task scores
// near 0, real work evolves through changing titles.
func (b Block) Entropy() float64 {
	if b.Samples == 0 {
		return 0
	}
	return float64(len(b.Titles)) / float64(b.Samples)
}

// Deep reports whether this is a deep-work block: ≥25 min in a productive app.
func (b Block) Deep() bool {
	return b.Category == Productive && b.Dur >= deepWorkMin
}

// DayMetrics are the sessionized view of one day's samples.
type DayMetrics struct {
	Blocks      []Block
	Active      time.Duration
	Idle        time.Duration
	Productive  time.Duration
	Neutral     time.Duration
	Distracting time.Duration
	DeepTotal   time.Duration
	DeepCount   int
	Longest     time.Duration // longest productive block
	Switches    int           // app changes within segments (glances included)
}

func (m DayMetrics) SwitchesPerHour() float64 {
	h := m.Active.Hours()
	if h == 0 {
		return 0
	}
	return float64(m.Switches) / h
}

// sessionize splits a day's samples into blocks and computes day metrics.
func sessionize(samples []Sample, pollSeconds int, rules []Rule) DayMetrics {
	tick := time.Duration(pollSeconds) * time.Second
	// Up to 2 missed samples (a delta of 3 ticks between recorded samples)
	// does not break a block; allow half a poll of scheduling jitter on top.
	maxGap := 3*tick + tick/2

	m := DayMetrics{}

	// Split into segments at idle samples and recording gaps; blocks never
	// span those.
	var segments [][]Sample
	var seg []Sample
	flush := func() {
		if len(seg) > 0 {
			segments = append(segments, seg)
			seg = nil
		}
	}
	for i, s := range samples {
		if s.Idle || s.App == "" {
			m.Idle += tick
			flush()
			continue
		}
		m.Active += tick
		switch classifySample(s, rules) {
		case Productive:
			m.Productive += tick
		case Distracting:
			m.Distracting += tick
		default:
			m.Neutral += tick
		}
		if len(seg) > 0 && s.TS.Sub(samples[i-1].TS) > maxGap {
			flush()
		}
		seg = append(seg, s)
	}
	flush()

	for _, seg := range segments {
		runs := splitRuns(seg)
		m.Switches += len(runs) - 1
		m.Blocks = append(m.Blocks, mergeGlances(runs, tick, rules)...)
	}

	for _, b := range m.Blocks {
		if b.Deep() {
			m.DeepTotal += b.Dur
			m.DeepCount++
		}
		if b.Category == Productive && b.Dur > m.Longest {
			m.Longest = b.Dur
		}
	}
	return m
}

// splitRuns breaks a segment into maximal same-app runs.
func splitRuns(seg []Sample) [][]Sample {
	var runs [][]Sample
	var run []Sample
	for _, s := range seg {
		if len(run) > 0 && s.App != run[0].App {
			runs = append(runs, run)
			run = nil
		}
		run = append(run, s)
	}
	if len(run) > 0 {
		runs = append(runs, run)
	}
	return runs
}

// mergeGlances turns runs into blocks, absorbing foreign interruptions of at
// most glanceSamples samples when the surrounding app is the same. Glance
// time counts toward the block's duration but not its titles.
func mergeGlances(runs [][]Sample, tick time.Duration, rules []Rule) []Block {
	var blocks []Block
	for i := 0; i < len(runs); {
		b := newBlock(runs[i], tick, rules)
		j := i + 1
		for j+1 < len(runs) && len(runs[j]) <= glanceSamples && runs[j+1][0].App == b.App {
			b.Dur += time.Duration(len(runs[j])) * tick // glance time
			extendBlock(&b, runs[j+1], tick)
			j += 2
		}
		blocks = append(blocks, b)
		i = j
	}
	return blocks
}

func newBlock(run []Sample, tick time.Duration, rules []Rule) Block {
	b := Block{App: run[0].App, Start: run[0].TS, Titles: map[string]int{}}
	// Block category: majority vote over its samples, so a chrome block on
	// youtube classifies by titles, not by the app being neutral.
	votes := map[Category]int{}
	for _, s := range run {
		votes[classifySample(s, rules)]++
	}
	b.Category = Neutral
	best := 0
	for _, cat := range []Category{Productive, Distracting, Neutral} {
		if votes[cat] > best {
			b.Category, best = cat, votes[cat]
		}
	}
	extendBlock(&b, run, tick)
	return b
}

func extendBlock(b *Block, run []Sample, tick time.Duration) {
	for _, s := range run {
		b.Titles[s.Title]++
	}
	b.Samples += len(run)
	b.Dur += time.Duration(len(run)) * tick
	b.End = run[len(run)-1].TS.Add(tick)
}
