package main

import "fmt"

// maxSwitchesPerHour is where the cohesion component bottoms out.
const maxSwitchesPerHour = 30.0

// lowEntropyThreshold flags deep-work blocks that loop one title — the
// farmer signature. Reported, never silently punished.
const lowEntropyThreshold = 0.05

// Score is the 0–100 daily score. Every component is a ratio or outcome —
// nothing in here grows by simply doing more (Goodhart).
type Score struct {
	Focus    float64 // productive / active
	Depth    float64 // deep-work / productive
	Cohesion float64 // 1 - min(1, switches-per-active-hour / 30)
	Yield    float64 // artifact-linked deep hours / deep hours
	Total    int
}

func computeScore(m DayMetrics, w Weights) Score {
	var s Score
	if m.Active == 0 {
		return s // nothing measured, nothing scored — cohesion isn't free
	}
	s.Focus = m.Productive.Seconds() / m.Active.Seconds()
	if m.Productive > 0 {
		s.Depth = m.DeepTotal.Seconds() / m.Productive.Seconds()
	}
	sw := m.SwitchesPerHour() / maxSwitchesPerHour
	if sw > 1 {
		sw = 1
	}
	s.Cohesion = 1 - sw
	s.Yield = yieldRatio(m.Blocks)

	total := s.Focus*float64(w.Focus) + s.Depth*float64(w.Depth) +
		s.Cohesion*float64(w.Cohesion) + s.Yield*float64(w.Yield)
	s.Total = int(total + 0.5)
	return s
}

// verdict is the one-line reading of the day, in the voice of the Fates.
func verdict(total int) string {
	switch {
	case total >= 80:
		return "the thread runs true"
	case total >= 60:
		return "a well-measured day"
	case total >= 40:
		return "the thread wanders"
	case total >= 20:
		return "frayed — little was woven"
	default:
		return "Atropos found nothing worth cutting"
	}
}

// farmerWarnings flags deep-work blocks bearing the farming signature:
// near-zero title entropy (looping one task).
func farmerWarnings(blocks []Block) []string {
	var warns []string
	for _, b := range blocks {
		if b.Deep() && b.Entropy() < lowEntropyThreshold {
			warns = append(warns, fmt.Sprintf(
				"%s %s–%s: title entropy %.3f — looping? low-entropy block",
				b.App, b.Start.Format("15:04"), b.End.Format("15:04"), b.Entropy()))
		}
	}
	return warns
}
