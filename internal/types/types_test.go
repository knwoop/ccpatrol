package types

import (
	"encoding/json"
	"testing"
)

func TestFindingRoundTrip(t *testing.T) {
	original := Finding{
		ID:           "F001",
		Severity:     "CRITICAL",
		Category:     "bug",
		File:         "main.go",
		LineStart:    10,
		LineEnd:      15,
		Title:        "Null pointer dereference",
		Explanation:  "Variable may be nil",
		SuggestedFix: "Add nil check before use",
		Confidence:   "HIGH",
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var decoded Finding
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if decoded != original {
		t.Errorf("round-trip mismatch:\n  got:  %+v\n  want: %+v", decoded, original)
	}
}

func TestFindingFalsePositiveOmitEmpty(t *testing.T) {
	f := Finding{ID: "F001", Severity: "LOW"}
	data, err := json.Marshal(f)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	// is_false_positive should be omitted when false
	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("unmarshal map: %v", err)
	}
	if _, ok := m["is_false_positive"]; ok {
		t.Error("is_false_positive should be omitted when false")
	}

	// When true, it should be present
	f.IsFalsePositive = true
	data, err = json.Marshal(f)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("unmarshal map: %v", err)
	}
	if _, ok := m["is_false_positive"]; !ok {
		t.Error("is_false_positive should be present when true")
	}
}

func TestReviewResultRoundTrip(t *testing.T) {
	original := ReviewResult{
		Findings: []Finding{
			{
				ID:           "F001",
				Severity:     "CRITICAL",
				Category:     "security",
				File:         "auth.go",
				LineStart:    42,
				LineEnd:      45,
				Title:        "SQL injection",
				Explanation:  "User input not sanitized",
				SuggestedFix: "Use parameterized query",
				Confidence:   "HIGH",
			},
			{
				ID:           "F002",
				Severity:     "LOW",
				Category:     "performance",
				File:         "handler.go",
				LineStart:    10,
				LineEnd:      10,
				Title:        "Unnecessary allocation",
				Explanation:  "Slice preallocated but unused",
				SuggestedFix: "Remove preallocation",
				Confidence:   "MEDIUM",
			},
		},
		Summary: ResultSummary{
			Total:     2,
			Critical:  1,
			Important: 0,
			Low:       1,
		},
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var decoded ReviewResult
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if len(decoded.Findings) != len(original.Findings) {
		t.Fatalf("findings count: got %d, want %d", len(decoded.Findings), len(original.Findings))
	}
	for i := range original.Findings {
		if decoded.Findings[i] != original.Findings[i] {
			t.Errorf("finding[%d] mismatch:\n  got:  %+v\n  want: %+v", i, decoded.Findings[i], original.Findings[i])
		}
	}
	if decoded.Summary != original.Summary {
		t.Errorf("summary mismatch:\n  got:  %+v\n  want: %+v", decoded.Summary, original.Summary)
	}
}

func TestEmptyReviewResult(t *testing.T) {
	original := ReviewResult{
		Findings: []Finding{},
		Summary:  ResultSummary{Total: 0},
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var decoded ReviewResult
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if len(decoded.Findings) != 0 {
		t.Errorf("expected empty findings, got %d", len(decoded.Findings))
	}
}
