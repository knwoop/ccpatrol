package schema

import (
	"strings"
	"testing"
)

func TestValidate_ValidFindings(t *testing.T) {
	input := `{
		"findings": [
			{
				"id": "F001",
				"severity": "CRITICAL",
				"category": "bug",
				"file": "main.go",
				"line_start": 10,
				"line_end": 20,
				"title": "Null pointer dereference",
				"explanation": "The pointer may be nil when accessed.",
				"suggested_fix": "Add a nil check before dereferencing.",
				"confidence": "HIGH"
			},
			{
				"id": "F002",
				"severity": "LOW",
				"category": "performance",
				"file": "utils.go",
				"line_start": 5,
				"line_end": 5,
				"title": "Inefficient string concatenation in loop",
				"explanation": "Using += in a loop creates many allocations.",
				"suggested_fix": "Use strings.Builder instead.",
				"confidence": "MEDIUM"
			}
		],
		"summary": {
			"total": 2,
			"critical": 1,
			"important": 0,
			"low": 1
		}
	}`

	result, err := Validate([]byte(input))
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if len(result.Findings) != 2 {
		t.Fatalf("expected 2 findings, got %d", len(result.Findings))
	}
	if result.Findings[0].ID != "F001" {
		t.Errorf("expected first finding ID F001, got %s", result.Findings[0].ID)
	}
	if result.Findings[0].Severity != "CRITICAL" {
		t.Errorf("expected severity CRITICAL, got %s", result.Findings[0].Severity)
	}
	if result.Findings[1].ID != "F002" {
		t.Errorf("expected second finding ID F002, got %s", result.Findings[1].ID)
	}
	if result.Summary.Total != 2 {
		t.Errorf("expected total 2, got %d", result.Summary.Total)
	}
	if result.Summary.Critical != 1 {
		t.Errorf("expected critical 1, got %d", result.Summary.Critical)
	}
	if result.Summary.Low != 1 {
		t.Errorf("expected low 1, got %d", result.Summary.Low)
	}
}

func TestValidate_MissingSummary(t *testing.T) {
	input := `{
		"findings": []
	}`

	_, err := Validate([]byte(input))
	if err == nil {
		t.Fatal("expected error for missing summary, got nil")
	}
	if !strings.Contains(err.Error(), "schema validation failed") {
		t.Errorf("expected schema validation error, got: %v", err)
	}
}

func TestValidate_InvalidSeverity(t *testing.T) {
	input := `{
		"findings": [
			{
				"id": "F001",
				"severity": "UNKNOWN",
				"category": "bug",
				"file": "main.go",
				"line_start": 1,
				"line_end": 1,
				"title": "Test",
				"explanation": "Test explanation",
				"suggested_fix": "Fix it",
				"confidence": "HIGH"
			}
		],
		"summary": {
			"total": 1,
			"critical": 0,
			"important": 0,
			"low": 0
		}
	}`

	_, err := Validate([]byte(input))
	if err == nil {
		t.Fatal("expected error for invalid severity, got nil")
	}
	if !strings.Contains(err.Error(), "schema validation failed") {
		t.Errorf("expected schema validation error, got: %v", err)
	}
}

func TestValidate_EmptySuggestedFix(t *testing.T) {
	input := `{
		"findings": [
			{
				"id": "F001",
				"severity": "CRITICAL",
				"category": "bug",
				"file": "main.go",
				"line_start": 1,
				"line_end": 1,
				"title": "Test",
				"explanation": "Test explanation",
				"suggested_fix": "",
				"confidence": "HIGH"
			}
		],
		"summary": {
			"total": 1,
			"critical": 1,
			"important": 0,
			"low": 0
		}
	}`

	_, err := Validate([]byte(input))
	if err == nil {
		t.Fatal("expected error for empty suggested_fix, got nil")
	}
	if !strings.Contains(err.Error(), "schema validation failed") {
		t.Errorf("expected schema validation error, got: %v", err)
	}
}

func TestValidate_BadIDPattern(t *testing.T) {
	input := `{
		"findings": [
			{
				"id": "X999",
				"severity": "LOW",
				"category": "bug",
				"file": "main.go",
				"line_start": 1,
				"line_end": 1,
				"title": "Test",
				"explanation": "Test explanation",
				"suggested_fix": "Fix it",
				"confidence": "HIGH"
			}
		],
		"summary": {
			"total": 1,
			"critical": 0,
			"important": 0,
			"low": 1
		}
	}`

	_, err := Validate([]byte(input))
	if err == nil {
		t.Fatal("expected error for bad id pattern, got nil")
	}
	if !strings.Contains(err.Error(), "schema validation failed") {
		t.Errorf("expected schema validation error, got: %v", err)
	}
}

func TestValidate_MalformedJSON(t *testing.T) {
	input := `{not valid json at all`

	_, err := Validate([]byte(input))
	if err == nil {
		t.Fatal("expected error for malformed JSON, got nil")
	}
	if !strings.Contains(err.Error(), "invalid JSON") {
		t.Errorf("expected invalid JSON error, got: %v", err)
	}
}

func TestValidate_EmptyFindingsWithZeroSummary(t *testing.T) {
	input := `{
		"findings": [],
		"summary": {
			"total": 0,
			"critical": 0,
			"important": 0,
			"low": 0
		}
	}`

	result, err := Validate([]byte(input))
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if len(result.Findings) != 0 {
		t.Errorf("expected 0 findings, got %d", len(result.Findings))
	}
	if result.Summary.Total != 0 {
		t.Errorf("expected total 0, got %d", result.Summary.Total)
	}
}
