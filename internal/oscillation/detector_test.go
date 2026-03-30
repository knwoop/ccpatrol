package oscillation

import (
	"testing"

	"github.com/knwoop/ccpatrol/internal/types"
)

func TestIsOscillating_NoHistory(t *testing.T) {
	f := types.Finding{File: "main.go", LineStart: 10, LineEnd: 20}
	if IsOscillating(f, nil) {
		t.Error("expected no oscillation with nil history")
	}
	if IsOscillating(f, []IterationSnapshot{}) {
		t.Error("expected no oscillation with empty history")
	}
}

func TestIsOscillating_OneIterationHistory(t *testing.T) {
	f := types.Finding{File: "main.go", LineStart: 10, LineEnd: 20}
	history := []IterationSnapshot{
		{Iteration: 1, Findings: []types.Finding{f}},
	}
	if IsOscillating(f, history) {
		t.Error("expected no oscillation with only 1 iteration of history")
	}
}

func TestIsOscillating_MatchesTwoIterationsAgo(t *testing.T) {
	// history has 2 iterations; current iteration is implicitly 3.
	// The finding in iteration 1 (N-2) has the same file and overlapping lines.
	past := types.Finding{File: "main.go", LineStart: 10, LineEnd: 20}
	history := []IterationSnapshot{
		{Iteration: 1, Findings: []types.Finding{past}},
		{Iteration: 2, Findings: []types.Finding{}},
	}
	current := types.Finding{File: "main.go", LineStart: 15, LineEnd: 25}
	if !IsOscillating(current, history) {
		t.Error("expected oscillation: same file, overlapping lines in N-2")
	}
}

func TestIsOscillating_SameFileNonOverlappingLines(t *testing.T) {
	past := types.Finding{File: "main.go", LineStart: 10, LineEnd: 20}
	history := []IterationSnapshot{
		{Iteration: 1, Findings: []types.Finding{past}},
		{Iteration: 2, Findings: []types.Finding{}},
	}
	current := types.Finding{File: "main.go", LineStart: 21, LineEnd: 30}
	if IsOscillating(current, history) {
		t.Error("expected no oscillation: same file but non-overlapping lines")
	}
}

func TestIsOscillating_DifferentFileSameLines(t *testing.T) {
	past := types.Finding{File: "main.go", LineStart: 10, LineEnd: 20}
	history := []IterationSnapshot{
		{Iteration: 1, Findings: []types.Finding{past}},
		{Iteration: 2, Findings: []types.Finding{}},
	}
	current := types.Finding{File: "other.go", LineStart: 10, LineEnd: 20}
	if IsOscillating(current, history) {
		t.Error("expected no oscillation: different file even though lines match")
	}
}

func TestIsOscillating_PartialOverlap(t *testing.T) {
	past := types.Finding{File: "main.go", LineStart: 10, LineEnd: 20}
	history := []IterationSnapshot{
		{Iteration: 1, Findings: []types.Finding{past}},
		{Iteration: 2, Findings: []types.Finding{}},
	}
	// Partial overlap: current starts at 20 which equals past's LineEnd.
	current := types.Finding{File: "main.go", LineStart: 20, LineEnd: 30}
	if !IsOscillating(current, history) {
		t.Error("expected oscillation: partial overlap on same file")
	}
}

func TestDetectAll(t *testing.T) {
	pastA := types.Finding{File: "a.go", LineStart: 1, LineEnd: 10}
	history := []IterationSnapshot{
		{Iteration: 1, Findings: []types.Finding{pastA}},
		{Iteration: 2, Findings: []types.Finding{}},
	}

	findingOsc := types.Finding{ID: "osc", File: "a.go", LineStart: 5, LineEnd: 15}
	findingClean := types.Finding{ID: "clean", File: "b.go", LineStart: 1, LineEnd: 5}

	clean, oscillating := DetectAll([]types.Finding{findingOsc, findingClean}, history)

	if len(clean) != 1 || clean[0].ID != "clean" {
		t.Errorf("expected 1 clean finding with ID 'clean', got %v", clean)
	}
	if len(oscillating) != 1 || oscillating[0].ID != "osc" {
		t.Errorf("expected 1 oscillating finding with ID 'osc', got %v", oscillating)
	}
}
