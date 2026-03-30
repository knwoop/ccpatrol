package steps

import (
	"context"
	"errors"
	"testing"

	"github.com/knwoop/ccpatrol/internal/llm"
	"github.com/knwoop/ccpatrol/internal/types"
)

// validateMockClient implements llm.Client for validate tests.
type validateMockClient struct {
	validateFn func(ctx context.Context, req llm.ValidateRequest) ([]byte, error)
}

func (m *validateMockClient) Review(ctx context.Context, req llm.ReviewRequest) ([]byte, error) {
	return nil, nil
}

func (m *validateMockClient) Fix(ctx context.Context, req llm.FixRequest) ([]byte, error) {
	return nil, nil
}

func (m *validateMockClient) Triage(ctx context.Context, req llm.TriageRequest) ([]byte, error) {
	return nil, nil
}

func (m *validateMockClient) Validate(ctx context.Context, req llm.ValidateRequest) ([]byte, error) {
	return m.validateFn(ctx, req)
}

func TestValidateStep_ZeroFindings(t *testing.T) {
	client := &validateMockClient{
		validateFn: func(ctx context.Context, req llm.ValidateRequest) ([]byte, error) {
			return emptyFindingsResponse(), nil
		},
	}

	result, err := ValidateStep(context.Background(), client, "test prompt", types.Config{})
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

func TestValidateStep_RemainingFindings(t *testing.T) {
	client := &validateMockClient{
		validateFn: func(ctx context.Context, req llm.ValidateRequest) ([]byte, error) {
			return validResponse(), nil
		},
	}

	result, err := ValidateStep(context.Background(), client, "test prompt", types.Config{})
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

func TestValidateStep_RetryOnSchemaError(t *testing.T) {
	calls := 0
	client := &validateMockClient{
		validateFn: func(ctx context.Context, req llm.ValidateRequest) ([]byte, error) {
			calls++
			if calls == 1 {
				return []byte(`not valid json`), nil
			}
			return validResponse(), nil
		},
	}

	result, err := ValidateStep(context.Background(), client, "test prompt", types.Config{})
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

func TestValidateStep_AllAttemptsFail(t *testing.T) {
	calls := 0
	client := &validateMockClient{
		validateFn: func(ctx context.Context, req llm.ValidateRequest) ([]byte, error) {
			calls++
			return []byte(`{"bad": "data"}`), nil
		},
	}

	_, err := ValidateStep(context.Background(), client, "test prompt", types.Config{})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if calls != 3 {
		t.Fatalf("expected 3 calls, got %d", calls)
	}
}

func TestValidateStep_ClientError(t *testing.T) {
	client := &validateMockClient{
		validateFn: func(ctx context.Context, req llm.ValidateRequest) ([]byte, error) {
			return nil, errors.New("network error")
		},
	}

	_, err := ValidateStep(context.Background(), client, "test prompt", types.Config{})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if got := err.Error(); got != "llm validate call failed: network error" {
		t.Errorf("unexpected error message: %s", got)
	}
}
