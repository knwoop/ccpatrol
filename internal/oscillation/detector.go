package oscillation

import "github.com/knwoop/ccpatrol/internal/types"

// IterationSnapshot captures the findings produced in a single review iteration.
type IterationSnapshot struct {
	Iteration int
	Findings  []types.Finding
}

// IsOscillating reports whether a finding is oscillating relative to the given
// history. A finding oscillates when a finding with the same file and
// overlapping line range appeared two iterations ago (iteration N-2, where N is
// the current iteration implied by len(history)+1). At least two prior
// iterations are required to make the comparison.
func IsOscillating(finding types.Finding, history []IterationSnapshot) bool {
	if len(history) < 2 {
		return false
	}

	// N-2 snapshot is the second-to-last element of history.
	nMinus2 := history[len(history)-2]

	for _, prev := range nMinus2.Findings {
		if prev.File == finding.File && linesOverlap(prev, finding) {
			return true
		}
	}
	return false
}

// DetectAll partitions findings into clean (non-oscillating) and oscillating
// lists based on the provided iteration history.
func DetectAll(findings []types.Finding, history []IterationSnapshot) (clean []types.Finding, oscillating []types.Finding) {
	for _, f := range findings {
		if IsOscillating(f, history) {
			oscillating = append(oscillating, f)
		} else {
			clean = append(clean, f)
		}
	}
	return clean, oscillating
}

// linesOverlap returns true when the line ranges of two findings overlap.
// Overlap condition: f1.LineStart <= f2.LineEnd && f2.LineStart <= f1.LineEnd
func linesOverlap(a, b types.Finding) bool {
	return a.LineStart <= b.LineEnd && b.LineStart <= a.LineEnd
}
