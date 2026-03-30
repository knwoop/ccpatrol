package steps

import (
	"context"
	"errors"
	"testing"

	"github.com/knwoop/ccpatrol/internal/llm"
	"github.com/knwoop/ccpatrol/internal/types"
)

// mockFixClient implements llm.Client for testing the Fix step.
type mockFixClient struct {
	fixFunc func(ctx context.Context, req llm.FixRequest) ([]byte, error)
	lastReq llm.FixRequest
}

func (m *mockFixClient) Review(_ context.Context, _ llm.ReviewRequest) ([]byte, error) {
	return nil, nil
}

func (m *mockFixClient) Fix(ctx context.Context, req llm.FixRequest) ([]byte, error) {
	m.lastReq = req
	return m.fixFunc(ctx, req)
}

func (m *mockFixClient) Triage(_ context.Context, _ llm.TriageRequest) ([]byte, error) {
	return nil, nil
}

func (m *mockFixClient) Validate(_ context.Context, _ llm.ValidateRequest) ([]byte, error) {
	return nil, nil
}

func TestFix_Success(t *testing.T) {
	client := &mockFixClient{
		fixFunc: func(_ context.Context, _ llm.FixRequest) ([]byte, error) {
			return []byte("ok"), nil
		},
	}

	cfg := types.Config{Verbose: true}
	err := Fix(context.Background(), client, "fix these issues", cfg)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
}

func TestFix_ClientError(t *testing.T) {
	client := &mockFixClient{
		fixFunc: func(_ context.Context, _ llm.FixRequest) ([]byte, error) {
			return nil, errors.New("llm unavailable")
		},
	}

	cfg := types.Config{Verbose: false}
	err := Fix(context.Background(), client, "fix prompt", cfg)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestFix_PassesPromptAndVerbose(t *testing.T) {
	client := &mockFixClient{
		fixFunc: func(_ context.Context, _ llm.FixRequest) ([]byte, error) {
			return []byte("done"), nil
		},
	}

	prompt := "please fix the code according to findings"
	cfg := types.Config{Verbose: true}

	err := Fix(context.Background(), client, prompt, cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if client.lastReq.PromptText != prompt {
		t.Errorf("PromptText = %q, want %q", client.lastReq.PromptText, prompt)
	}
	if client.lastReq.Verbose != true {
		t.Errorf("Verbose = %v, want true", client.lastReq.Verbose)
	}
}
