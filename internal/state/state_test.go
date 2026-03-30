package state

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/knwoop/ccpatrol/internal/types"
)

func TestLoad_NonExistentFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "does-not-exist.json")

	s, err := Load(path)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if s == nil {
		t.Fatal("expected non-nil state")
	}
	if s.Iteration != 0 {
		t.Errorf("expected Iteration 0, got %d", s.Iteration)
	}
	if len(s.History) != 0 {
		t.Errorf("expected empty History, got %d entries", len(s.History))
	}
	if len(s.Directives) != 0 {
		t.Errorf("expected empty Directives, got %d entries", len(s.Directives))
	}
}

func TestSaveThenLoad_RoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "state.json")

	original := &State{
		Iteration:     2,
		MaxIterations: 5,
		BaseBranch:    "main",
		VerifyConfig: VerifyConfig{
			TestCmd:      "go test ./...",
			LintCmd:      "golangci-lint run",
			TypecheckCmd: "go vet ./...",
		},
		Directives: []Directive{
			{
				File:           "cmd/main.go",
				Lines:          "10-20",
				LockedApproach: "use context cancellation",
				LockedAtIter:   1,
				Reason:         "clean shutdown",
			},
		},
		History: []IterationRecord{
			{
				Iteration:   1,
				FindingsIDs: []string{"f1", "f2"},
				FixedIDs:    []string{"f1"},
				RemainingIDs: []string{"f2"},
				DroppedIDs:  []string{},
				VerifyPassed: true,
				VerifyFailures: []string{},
				FindingsSnapshot: []types.Finding{
					{
						ID:          "f1",
						Severity:    "critical",
						Category:    "security",
						File:        "cmd/main.go",
						LineStart:   10,
						LineEnd:     15,
						Title:       "SQL injection",
						Explanation: "user input not sanitized",
						SuggestedFix: "use parameterized queries",
						Confidence:  "high",
					},
				},
			},
			{
				Iteration:      2,
				FindingsIDs:    []string{"f2"},
				FixedIDs:       []string{"f2"},
				RemainingIDs:   []string{},
				DroppedIDs:     []string{},
				VerifyPassed:   false,
				VerifyFailures: []string{"lint: unused variable"},
				FindingsSnapshot: []types.Finding{
					{
						ID:              "f2",
						Severity:        "low",
						Category:        "style",
						File:            "pkg/util.go",
						LineStart:       5,
						LineEnd:         5,
						Title:           "unused variable",
						Explanation:     "declared but not used",
						SuggestedFix:    "remove variable",
						Confidence:      "high",
						IsFalsePositive: true,
					},
				},
			},
		},
	}

	if err := original.Save(path); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	loaded, err := Load(path)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if !reflect.DeepEqual(original, loaded) {
		t.Errorf("round-trip mismatch\noriginal: %+v\nloaded:   %+v", original, loaded)
	}
}

func TestAddIteration_IncrementsCounter(t *testing.T) {
	s := &State{}

	if s.Iteration != 0 {
		t.Fatalf("expected initial Iteration 0, got %d", s.Iteration)
	}

	s.AddIteration(IterationRecord{
		Iteration:   1,
		FindingsIDs: []string{"f1"},
		FixedIDs:    []string{"f1"},
	})

	if s.Iteration != 1 {
		t.Errorf("expected Iteration 1 after first AddIteration, got %d", s.Iteration)
	}
}

func TestAddIteration_MultipleCallsBuildHistory(t *testing.T) {
	s := &State{}

	records := []IterationRecord{
		{
			Iteration:    1,
			FindingsIDs:  []string{"f1", "f2"},
			FixedIDs:     []string{"f1"},
			RemainingIDs: []string{"f2"},
			VerifyPassed: true,
		},
		{
			Iteration:    2,
			FindingsIDs:  []string{"f2"},
			FixedIDs:     []string{"f2"},
			RemainingIDs: []string{},
			VerifyPassed: true,
		},
		{
			Iteration:    3,
			FindingsIDs:  []string{"f3"},
			FixedIDs:     []string{},
			RemainingIDs: []string{"f3"},
			DroppedIDs:   []string{"f3"},
			VerifyPassed: false,
			VerifyFailures: []string{"test failed"},
		},
	}

	for _, r := range records {
		s.AddIteration(r)
	}

	if s.Iteration != 3 {
		t.Errorf("expected Iteration 3, got %d", s.Iteration)
	}
	if len(s.History) != 3 {
		t.Fatalf("expected 3 history entries, got %d", len(s.History))
	}

	for i, r := range records {
		if !reflect.DeepEqual(s.History[i], r) {
			t.Errorf("history[%d] mismatch\nexpected: %+v\ngot:      %+v", i, r, s.History[i])
		}
	}
}

func TestSave_AtomicWrite(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "state.json")

	s := &State{
		Iteration:     1,
		MaxIterations: 10,
		BaseBranch:    "develop",
	}

	if err := s.Save(path); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	// The file should exist at the target path.
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("expected file at %s, got error: %v", path, err)
	}
	if info.Size() == 0 {
		t.Error("expected non-empty file")
	}

	// No leftover temp files should remain in the directory.
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("ReadDir failed: %v", err)
	}
	for _, e := range entries {
		if e.Name() != "state.json" {
			t.Errorf("unexpected leftover file: %s", e.Name())
		}
	}
}
