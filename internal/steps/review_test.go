package steps

import (
	"context"
	"errors"
	"testing"

	"github.com/knwoop/ccpatrol/internal/llm"
	"github.com/knwoop/ccpatrol/internal/types"
)

// mockClient implements llm.Client for testing.
type mockClient struct {
	reviewFn func(ctx context.Context, req llm.ReviewRequest) ([]byte, error)
}

func (m *mockClient) Review(ctx context.Context, req llm.ReviewRequest) ([]byte, error) {
	return m.reviewFn(ctx, req)
}

func (m *mockClient) Fix(ctx context.Context, req llm.FixRequest) ([]byte, error) {
	return nil, nil
}

func (m *mockClient) Triage(ctx context.Context, req llm.TriageRequest) ([]byte, error) {
	return nil, nil
}

func (m *mockClient) Validate(ctx context.Context, req llm.ValidateRequest) ([]byte, error) {
	return nil, nil
}

// validResponse returns a valid JSON response with one finding.
func validResponse() []byte {
	return []byte(`{
		"findings": [
			{
				"id": "F001",
				"severity": "CRITICAL",
				"category": "bug",
				"file": "main.go",
				"line_start": 10,
				"line_end": 12,
				"title": "Null pointer dereference",
				"explanation": "The pointer is not checked before use.",
				"suggested_fix": "Add a nil check before dereferencing.",
				"confidence": "HIGH"
			}
		],
		"summary": {
			"total": 1,
			"critical": 1,
			"important": 0,
			"low": 0
		}
	}`)
}

// emptyFindingsResponse returns a valid JSON response with zero findings.
func emptyFindingsResponse() []byte {
	return []byte(`{
		"findings": [],
		"summary": {
			"total": 0,
			"critical": 0,
			"important": 0,
			"low": 0
		}
	}`)
}

func TestReview_ValidResponse(t *testing.T) {
	client := &mockClient{
		reviewFn: func(ctx context.Context, req llm.ReviewRequest) ([]byte, error) {
			return validResponse(), nil
		},
	}

	result, err := Review(context.Background(), client, "test prompt", types.Config{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Findings) != 1 {
		t.Fatalf("expected 1 finding, got %d", len(result.Findings))
	}
	if result.Findings[0].ID != "F001" {
		t.Errorf("expected finding ID F001, got %s", result.Findings[0].ID)
	}
	if result.Summary.Critical != 1 {
		t.Errorf("expected 1 critical, got %d", result.Summary.Critical)
	}
}

func TestReview_InvalidThenValid(t *testing.T) {
	calls := 0
	client := &mockClient{
		reviewFn: func(ctx context.Context, req llm.ReviewRequest) ([]byte, error) {
			calls++
			if calls == 1 {
				return []byte(`not valid json`), nil
			}
			return validResponse(), nil
		},
	}

	result, err := Review(context.Background(), client, "test prompt", types.Config{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if calls != 2 {
		t.Fatalf("expected 2 calls, got %d", calls)
	}
	if len(result.Findings) != 1 {
		t.Fatalf("expected 1 finding, got %d", len(result.Findings))
	}
}

func TestReview_AllAttemptsInvalid(t *testing.T) {
	calls := 0
	client := &mockClient{
		reviewFn: func(ctx context.Context, req llm.ReviewRequest) ([]byte, error) {
			calls++
			return []byte(`{"bad": "data"}`), nil
		},
	}

	_, err := Review(context.Background(), client, "test prompt", types.Config{})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if calls != 3 {
		t.Fatalf("expected 3 calls, got %d", calls)
	}
}

func TestReview_ZeroFindings(t *testing.T) {
	client := &mockClient{
		reviewFn: func(ctx context.Context, req llm.ReviewRequest) ([]byte, error) {
			return emptyFindingsResponse(), nil
		},
	}

	result, err := Review(context.Background(), client, "test prompt", types.Config{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Findings) != 0 {
		t.Fatalf("expected 0 findings, got %d", len(result.Findings))
	}
	if result.Summary.Total != 0 {
		t.Errorf("expected total 0, got %d", result.Summary.Total)
	}
}

func TestReview_ClientError(t *testing.T) {
	client := &mockClient{
		reviewFn: func(ctx context.Context, req llm.ReviewRequest) ([]byte, error) {
			return nil, errors.New("network error")
		},
	}

	_, err := Review(context.Background(), client, "test prompt", types.Config{})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if got := err.Error(); got != "llm review call failed: network error" {
		t.Errorf("unexpected error message: %s", got)
	}
}
