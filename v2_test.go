package main

import (
	"fmt"
	"testing"
	"time"
)

var t0 = time.Date(2026, 7, 12, 9, 0, 0, 0, time.Local)

// gen emits n samples 5s apart starting at start; titleFn varies the title.
func gen(start time.Time, app string, n int, titleFn func(i int) string) []Sample {
	out := make([]Sample, n)
	for i := range out {
		out[i] = Sample{TS: start.Add(time.Duration(i) * 5 * time.Second), App: app, Title: titleFn(i)}
	}
	return out
}

func fixed(title string) func(int) string { return func(int) string { return title } }

func after(samples []Sample) time.Time {
	return samples[len(samples)-1].TS.Add(5 * time.Second)
}

func TestRulePrecedence(t *testing.T) {
	rules := defaultRules()
	cases := []struct {
		app, title string
		want       Category
	}{
		{"chrome", "cat videos - YouTube", Distracting}, // title beats app
		{"chrome", "lacheisis - GitHub", Productive},
		{"chrome", "weather forecast", Neutral}, // unmatched
		{"code", "main.go - lacheisis", Productive},
		{"code", "watching YouTube in an editor tab", Distracting}, // title pass runs first
	}
	for _, c := range cases {
		got := classifySample(Sample{App: c.app, Title: c.title}, rules)
		if got != c.want {
			t.Errorf("classify(%s, %q) = %s, want %s", c.app, c.title, got, c.want)
		}
	}
}

func TestGlanceTolerance(t *testing.T) {
	rules := defaultRules()
	work := gen(t0, "code", 100, fixed("main.go"))
	glance := gen(after(work), "chrome", 2, fixed("docs"))
	more := gen(after(glance), "code", 100, fixed("main.go"))
	m := sessionize(append(append(work, glance...), more...), 5, rules)
	if len(m.Blocks) != 1 {
		t.Fatalf("2-sample glance should not break the block, got %d blocks", len(m.Blocks))
	}
	if m.Switches != 2 {
		t.Errorf("glance still counts as switches, got %d want 2", m.Switches)
	}

	interrupt := gen(after(work), "chrome", 3, fixed("docs"))
	more = gen(after(interrupt), "code", 100, fixed("main.go"))
	m = sessionize(append(append(work, interrupt...), more...), 5, rules)
	if len(m.Blocks) != 3 {
		t.Errorf("3-sample interruption should break the block, got %d blocks", len(m.Blocks))
	}
}

func TestIdleAndGapBreakBlocks(t *testing.T) {
	rules := defaultRules()
	a := gen(t0, "code", 50, fixed("main.go"))
	idle := []Sample{{TS: after(a), Idle: true}}
	b := gen(after(a).Add(5*time.Second), "code", 50, fixed("main.go"))
	m := sessionize(append(append(a, idle...), b...), 5, rules)
	if len(m.Blocks) != 2 {
		t.Errorf("idle sample must break blocks, got %d", len(m.Blocks))
	}

	// 10s gap (2 missed samples) tolerated, 20s gap breaks
	c := gen(after(a).Add(10*time.Second), "code", 50, fixed("main.go"))
	m = sessionize(append(a, c...), 5, rules)
	if len(m.Blocks) != 1 {
		t.Errorf("10s gap should not break block, got %d", len(m.Blocks))
	}
	d := gen(after(a).Add(20*time.Second), "code", 50, fixed("main.go"))
	m = sessionize(append(a, d...), 5, rules)
	if len(m.Blocks) != 2 {
		t.Errorf("20s gap must break block, got %d", len(m.Blocks))
	}
}

func TestEntropy(t *testing.T) {
	rules := defaultRules()
	loop := sessionize(gen(t0, "code", 400, fixed("same task")), 5, rules)
	varied := sessionize(gen(t0, "code", 400, func(i int) string {
		return fmt.Sprintf("file%d.go - lacheisis", i/20)
	}), 5, rules)
	if e := loop.Blocks[0].Entropy(); e >= lowEntropyThreshold {
		t.Errorf("looping block entropy %.3f should be below %.2f", e, lowEntropyThreshold)
	}
	if e := varied.Blocks[0].Entropy(); e < lowEntropyThreshold {
		t.Errorf("varied block entropy %.3f should be above %.2f", e, lowEntropyThreshold)
	}
}

func TestCommitAttribution(t *testing.T) {
	rules := defaultRules()
	work := gen(t0, "code", 400, fixed("main.go")) // 33m20s deep block
	m := sessionize(work, 5, rules)
	if len(m.Blocks) != 1 || !m.Blocks[0].Deep() {
		t.Fatalf("expected one deep block")
	}
	end := m.Blocks[0].End
	commits := []Commit{
		{Hash: "a", TS: t0.Add(10 * time.Minute)},   // inside
		{Hash: "b", TS: end.Add(10 * time.Minute)},  // within 30m after
		{Hash: "c", TS: end.Add(40 * time.Minute)},  // too late — orphan
		{Hash: "d", TS: t0.Add(-10 * time.Minute)},  // before — orphan
	}
	orphans := attributeCommits(m.Blocks, commits)
	if len(m.Blocks[0].Commits) != 2 {
		t.Errorf("block should get commits a,b — got %d", len(m.Blocks[0].Commits))
	}
	if len(orphans) != 2 {
		t.Errorf("expected 2 orphans, got %d", len(orphans))
	}
	if yieldRatio(m.Blocks) != 1 {
		t.Errorf("single linked deep block should yield 1, got %.2f", yieldRatio(m.Blocks))
	}
}

// TestFarmerVsDeepWorker is the anti-gaming acceptance test: the farmer sits
// 6 uniform hours in one app with one looping title and no commits; the deep
// worker does less total time across three evolving blocks with commits. The
// farmer must score materially lower despite more "productive" hours.
func TestFarmerVsDeepWorker(t *testing.T) {
	rules := defaultRules()
	w := defaultWeights()

	farmer := gen(t0, "code", 6*720, fixed("kiro board")) // 6h, one title
	fm := sessionize(farmer, 5, rules)
	attributeCommits(fm.Blocks, nil)
	farmerScore := computeScore(fm, w)

	blockA := gen(t0, "code", 720, func(i int) string { return fmt.Sprintf("blocks.go:%d", i/10) }) // 1h
	browse := gen(after(blockA), "chrome", 60, fixed("weather forecast"))                           // 5m neutral
	blockB := gen(after(browse), "code", 540, func(i int) string { return fmt.Sprintf("report.go:%d", i/10) }) // 45m
	pause := []Sample{{TS: after(blockB), Idle: true}}
	blockC := gen(after(blockB).Add(10*time.Minute), "windowsterminal", 360, func(i int) string {
		return fmt.Sprintf("go test run %d", i/10)
	}) // 30m
	deep := append(append(append(append(blockA, browse...), blockB...), pause...), blockC...)
	dm := sessionize(deep, 5, rules)
	dmA, dmB := dm.Blocks[0], dm.Blocks[2]
	attributeCommits(dm.Blocks, []Commit{
		{Hash: "a", TS: dmA.End.Add(2 * time.Minute), Files: 3},
		{Hash: "b", TS: dmB.Start.Add(20 * time.Minute), Files: 5},
	})
	deepScore := computeScore(dm, w)

	if farmerScore.Total >= deepScore.Total-10 {
		t.Errorf("farmer (%d) must score materially below deep worker (%d)",
			farmerScore.Total, deepScore.Total)
	}
	if len(farmerWarnings(fm.Blocks)) == 0 {
		t.Error("farmer's looping block must carry a low-entropy warning")
	}
	if len(farmerWarnings(dm.Blocks)) != 0 {
		t.Error("deep worker should carry no low-entropy warnings")
	}
}

func TestScoreBounds(t *testing.T) {
	w := defaultWeights()
	if s := computeScore(DayMetrics{}, w); s.Total != 0 {
		t.Errorf("empty day should score 0, got %d", s.Total)
	}
	perfect := DayMetrics{
		Active: 4 * time.Hour, Productive: 4 * time.Hour, DeepTotal: 4 * time.Hour,
		Blocks: []Block{{App: "code", Category: Productive, Dur: 4 * time.Hour,
			Commits: []Commit{{Hash: "x"}}}},
	}
	if s := computeScore(perfect, w); s.Total > 100 || s.Total < 99 {
		t.Errorf("perfect day should score ~100, got %d", s.Total)
	}
}
