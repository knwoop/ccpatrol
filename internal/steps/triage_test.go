package steps

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/knwoop/ccpatrol/internal/llm"
	"github.com/knwoop/ccpatrol/internal/state"
	"github.com/knwoop/ccpatrol/internal/types"
)

// mockTriageClient implements llm.Client for triage tests.
type mockTriageClient struct {
	triageResponses [][]byte // successive Triage responses (one per call)
	triageErrors    []error
	callCount       int
}

func (m *mockTriageClient) Review(_ context.Context, _ llm.ReviewRequest) ([]byte, error) {
	return nil, errors.New("Review not implemented in mock")
}

func (m *mockTriageClient) Fix(_ context.Context, _ llm.FixRequest) ([]byte, error) {
	return nil, errors.New("Fix not implemented in mock")
}

func (m *mockTriageClient) Triage(_ context.Context, _ llm.TriageRequest) ([]byte, error) {
	idx := m.callCount
	m.callCount++
	if idx < len(m.triageErrors) && m.triageErrors[idx] != nil {
		return nil, m.triageErrors[idx]
	}
	if idx < len(m.triageResponses) {
		return m.triageResponses[idx], nil
	}
	return nil, errors.New("no more mock triage responses")
}

func (m *mockTriageClient) Validate(_ context.Context, _ llm.ValidateRequest) ([]byte, error) {
	return nil, errors.New("Validate not implemented in mock")
}

// helper to build a valid schema-compliant ReviewResult JSON.
func mustMarshalResult(t *testing.T, findings []types.Finding) []byte {
	t.Helper()
	summary := types.ResultSummary{Total: len(findings)}
	for _, f := range findings {
		switch f.Severity {
		case "CRITICAL":
			summary.Critical++
		case "IMPORTANT":
			summary.Important++
		case "LOW":
			summary.Low++
		}
	}
	rr := types.ReviewResult{Findings: findings, Summary: summary}
	data, err := json.Marshal(rr)
	if err != nil {
		t.Fatalf("mustMarshalResult: %v", err)
	}
	return data
}

func baseCfg() types.Config {
	return types.Config{}
}

func baseFinding(id, severity, confidence, file string) types.Finding {
	return types.Finding{
		ID:           id,
		Severity:     severity,
		Category:     "bug",
		File:         file,
		LineStart:    1,
		LineEnd:      5,
		Title:        "Test finding " + id,
		Explanation:  "explanation",
		SuggestedFix: "fix it",
		Confidence:   confidence,
	}
}

// Test 1: LLM downgrades IMPORTANT -> LOW -> Go drops it.
func TestTriageLLMDowngradesToLow(t *testing.T) {
	original := baseFinding("F001", "IMPORTANT", "HIGH", "main.go")

	// LLM returns with severity downgraded to LOW.
	downgraded := original
	downgraded.Severity = "LOW"

	mock := &mockTriageClient{
		triageResponses: [][]byte{mustMarshalResult(t, []types.Finding{downgraded})},
	}

	result, err := Triage(
		context.Background(), mock,
		[]types.Finding{original},
		&state.State{},
		"file contents", []string{"main.go"},
		"triage prompt", baseCfg(),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result.Actionable) != 0 {
		t.Errorf("expected 0 actionable, got %d", len(result.Actionable))
	}
	if len(result.DroppedLow) != 1 {
		t.Errorf("expected 1 dropped-low, got %d", len(result.DroppedLow))
	}
}

// Test 2: LLM marks finding as false positive -> Go drops it.
func TestTriageFalsePositive(t *testing.T) {
	original := baseFinding("F001", "CRITICAL", "HIGH", "main.go")

	fp := original
	fp.IsFalsePositive = true

	mock := &mockTriageClient{
		triageResponses: [][]byte{mustMarshalResult(t, []types.Finding{fp})},
	}

	result, err := Triage(
		context.Background(), mock,
		[]types.Finding{original},
		&state.State{},
		"file contents", []string{"main.go"},
		"triage prompt", baseCfg(),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result.Actionable) != 0 {
		t.Errorf("expected 0 actionable, got %d", len(result.Actionable))
	}
	if len(result.DroppedFP) != 1 {
		t.Errorf("expected 1 dropped-FP, got %d", len(result.DroppedFP))
	}
}

// Test 3: Finding on file outside diffFiles -> Go drops it.
func TestTriageFileOutsideDiff(t *testing.T) {
	original := baseFinding("F001", "CRITICAL", "HIGH", "other.go")

	mock := &mockTriageClient{
		triageResponses: [][]byte{mustMarshalResult(t, []types.Finding{original})},
	}

	result, err := Triage(
		context.Background(), mock,
		[]types.Finding{original},
		&state.State{},
		"file contents", []string{"main.go"}, // other.go is NOT in diffFiles
		"triage prompt", baseCfg(),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result.Actionable) != 0 {
		t.Errorf("expected 0 actionable, got %d", len(result.Actionable))
	}
	if len(result.DroppedLow) != 1 {
		t.Errorf("expected 1 dropped (outside diff), got %d", len(result.DroppedLow))
	}
}

// Test 4: Oscillating finding -> dropped + directive created.
func TestTriageOscillatingFinding(t *testing.T) {
	f := baseFinding("F001", "CRITICAL", "HIGH", "main.go")

	// Build history with 2 prior iterations. Iteration N-2 has same file + overlapping lines.
	st := &state.State{
		Iteration: 3,
		History: []state.IterationRecord{
			{
				Iteration: 1,
				FindingsSnapshot: []types.Finding{
					baseFinding("F001", "CRITICAL", "HIGH", "main.go"),
				},
			},
			{
				Iteration: 2,
				FindingsSnapshot: []types.Finding{
					baseFinding("F002", "IMPORTANT", "HIGH", "other.go"),
				},
			},
		},
	}

	mock := &mockTriageClient{
		triageResponses: [][]byte{mustMarshalResult(t, []types.Finding{f})},
	}

	result, err := Triage(
		context.Background(), mock,
		[]types.Finding{f},
		st,
		"file contents", []string{"main.go"},
		"triage prompt", baseCfg(),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result.Actionable) != 0 {
		t.Errorf("expected 0 actionable, got %d", len(result.Actionable))
	}
	if len(result.Oscillating) != 1 {
		t.Errorf("expected 1 oscillating, got %d", len(result.Oscillating))
	}
	if len(result.NewDirectives) != 1 {
		t.Fatalf("expected 1 directive, got %d", len(result.NewDirectives))
	}
	d := result.NewDirectives[0]
	if d.File != "main.go" {
		t.Errorf("directive file: got %q, want %q", d.File, "main.go")
	}
	if d.LockedAtIter != 3 {
		t.Errorf("directive locked_at_iter: got %d, want 3", d.LockedAtIter)
	}
	if d.Reason != "oscillation detected" {
		t.Errorf("directive reason: got %q, want %q", d.Reason, "oscillation detected")
	}
}

// Test 5: All CRITICAL/HIGH findings pass through.
func TestTriageCriticalAndHighPassThrough(t *testing.T) {
	findings := []types.Finding{
		baseFinding("F001", "CRITICAL", "HIGH", "main.go"),
		baseFinding("F002", "IMPORTANT", "HIGH", "main.go"),
		baseFinding("F003", "IMPORTANT", "MEDIUM", "main.go"),
	}

	mock := &mockTriageClient{
		triageResponses: [][]byte{mustMarshalResult(t, findings)},
	}

	result, err := Triage(
		context.Background(), mock,
		findings,
		&state.State{},
		"file contents", []string{"main.go"},
		"triage prompt", baseCfg(),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result.Actionable) != 3 {
		t.Errorf("expected 3 actionable, got %d", len(result.Actionable))
	}
	if len(result.DroppedLow) != 0 {
		t.Errorf("expected 0 dropped-low, got %d", len(result.DroppedLow))
	}
	if len(result.DroppedFP) != 0 {
		t.Errorf("expected 0 dropped-FP, got %d", len(result.DroppedFP))
	}
}

// Test 6: Mixed findings: various drop reasons, some pass through.
func TestTriageMixedFindings(t *testing.T) {
	// F001: CRITICAL, HIGH, in diff -> actionable
	f1 := baseFinding("F001", "CRITICAL", "HIGH", "main.go")

	// F002: LLM marks as false positive -> dropped-FP
	f2 := baseFinding("F002", "IMPORTANT", "HIGH", "main.go")
	f2.IsFalsePositive = true

	// F003: LOW severity -> dropped-low
	f3 := baseFinding("F003", "LOW", "HIGH", "main.go")

	// F004: LOW confidence -> dropped-conf
	f4 := baseFinding("F004", "IMPORTANT", "LOW", "main.go")

	// F005: file not in diff -> dropped-low
	f5 := baseFinding("F005", "CRITICAL", "HIGH", "notindiff.go")

	// F006: IMPORTANT, MEDIUM, in diff -> actionable
	f6 := baseFinding("F006", "IMPORTANT", "MEDIUM", "main.go")

	mock := &mockTriageClient{
		triageResponses: [][]byte{mustMarshalResult(t, []types.Finding{f1, f2, f3, f4, f5, f6})},
	}

	result, err := Triage(
		context.Background(), mock,
		[]types.Finding{f1, f2, f3, f4, f5, f6},
		&state.State{},
		"file contents", []string{"main.go"},
		"triage prompt", baseCfg(),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result.Actionable) != 2 {
		t.Errorf("expected 2 actionable (F001, F006), got %d", len(result.Actionable))
	}
	if len(result.DroppedFP) != 1 {
		t.Errorf("expected 1 dropped-FP (F002), got %d", len(result.DroppedFP))
	}
	if len(result.DroppedLow) != 2 {
		t.Errorf("expected 2 dropped-low (F003+F005), got %d", len(result.DroppedLow))
	}
	if len(result.DroppedConf) != 1 {
		t.Errorf("expected 1 dropped-conf (F004), got %d", len(result.DroppedConf))
	}
}

// Test 7: LLM schema failure on all retries -> error.
func TestTriageSchemaFailureAllRetries(t *testing.T) {
	badJSON := []byte(`{"not_valid": true}`)

	mock := &mockTriageClient{
		triageResponses: [][]byte{badJSON, badJSON, badJSON},
	}

	_, err := Triage(
		context.Background(), mock,
		[]types.Finding{baseFinding("F001", "CRITICAL", "HIGH", "main.go")},
		&state.State{},
		"file contents", []string{"main.go"},
		"triage prompt", baseCfg(),
	)
	if err == nil {
		t.Fatal("expected error after all retries exhausted")
	}
	if !strings.Contains(err.Error(), "triage failed after 3 attempts") {
		t.Errorf("unexpected error message: %v", err)
	}
	if mock.callCount != 3 {
		t.Errorf("expected 3 LLM calls, got %d", mock.callCount)
	}
}
