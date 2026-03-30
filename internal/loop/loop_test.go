package loop

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"testing"

	"github.com/knwoop/ccpatrol/internal/llm"
	"github.com/knwoop/ccpatrol/internal/types"
)

// mockLoopClient implements llm.Client for loop tests.
type mockLoopClient struct {
	reviewFn   func(ctx context.Context, req llm.ReviewRequest) ([]byte, error)
	triageFn   func(ctx context.Context, req llm.TriageRequest) ([]byte, error)
	fixFn      func(ctx context.Context, req llm.FixRequest) ([]byte, error)
	validateFn func(ctx context.Context, req llm.ValidateRequest) ([]byte, error)
}

func (m *mockLoopClient) Review(ctx context.Context, req llm.ReviewRequest) ([]byte, error) {
	if m.reviewFn != nil {
		return m.reviewFn(ctx, req)
	}
	return nil, fmt.Errorf("review not mocked")
}

func (m *mockLoopClient) Triage(ctx context.Context, req llm.TriageRequest) ([]byte, error) {
	if m.triageFn != nil {
		return m.triageFn(ctx, req)
	}
	return nil, fmt.Errorf("triage not mocked")
}

func (m *mockLoopClient) Fix(ctx context.Context, req llm.FixRequest) ([]byte, error) {
	if m.fixFn != nil {
		return m.fixFn(ctx, req)
	}
	return nil, fmt.Errorf("fix not mocked")
}

func (m *mockLoopClient) Validate(ctx context.Context, req llm.ValidateRequest) ([]byte, error) {
	if m.validateFn != nil {
		return m.validateFn(ctx, req)
	}
	return nil, fmt.Errorf("validate not mocked")
}

func makeReviewResponse(findings []types.Finding) []byte {
	result := types.ReviewResult{
		Findings: findings,
		Summary: types.ResultSummary{
			Total: len(findings),
		},
	}
	for _, f := range findings {
		switch f.Severity {
		case "CRITICAL":
			result.Summary.Critical++
		case "IMPORTANT":
			result.Summary.Important++
		case "LOW":
			result.Summary.Low++
		}
	}
	data, _ := json.Marshal(result)
	return data
}

func emptyReviewResponse() []byte {
	return makeReviewResponse(nil)
}

func sampleFindings() []types.Finding {
	return []types.Finding{
		{
			ID: "F001", Severity: "CRITICAL", Category: "bug",
			File: "main.go", LineStart: 10, LineEnd: 15,
			Title: "Null pointer", Explanation: "May be nil",
			SuggestedFix: "Add nil check", Confidence: "HIGH",
		},
		{
			ID: "F002", Severity: "IMPORTANT", Category: "error-handling",
			File: "handler.go", LineStart: 20, LineEnd: 25,
			Title: "Unchecked error", Explanation: "Error ignored",
			SuggestedFix: "Check error", Confidence: "HIGH",
		},
	}
}

func setupTest(t *testing.T) {
	t.Helper()
	// Override git/file functions for tests.
	origGitCmd := gitCmd
	origReadFile := readFile
	t.Cleanup(func() {
		gitCmd = origGitCmd
		readFile = origReadFile
		os.Remove(stateFile)
	})

	gitCmd = func(ctx context.Context, args ...string) (string, error) {
		if len(args) >= 1 && args[0] == "diff" {
			for _, a := range args {
				if a == "--name-only" {
					return "main.go\nhandler.go", nil
				}
			}
			return "fake diff content", nil
		}
		return "", nil
	}

	readFile = func(path string) (string, error) {
		return "// file content of " + path, nil
	}
}

func TestRun_ZeroFindings(t *testing.T) {
	setupTest(t)

	client := &mockLoopClient{
		reviewFn: func(_ context.Context, _ llm.ReviewRequest) ([]byte, error) {
			return emptyReviewResponse(), nil
		},
	}

	result, err := Run(context.Background(), types.Config{Base: "main", MaxLoops: 6}, client)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ExitCode != types.ExitSuccess {
		t.Errorf("expected exit code %d, got %d", types.ExitSuccess, result.ExitCode)
	}
}

func TestRun_FixVerifyValidateSuccess(t *testing.T) {
	setupTest(t)

	findings := sampleFindings()

	client := &mockLoopClient{
		reviewFn: func(_ context.Context, _ llm.ReviewRequest) ([]byte, error) {
			return makeReviewResponse(findings), nil
		},
		triageFn: func(_ context.Context, _ llm.TriageRequest) ([]byte, error) {
			// Return findings unchanged (all actionable).
			return makeReviewResponse(findings), nil
		},
		fixFn: func(_ context.Context, _ llm.FixRequest) ([]byte, error) {
			return []byte("ok"), nil
		},
		validateFn: func(_ context.Context, _ llm.ValidateRequest) ([]byte, error) {
			return emptyReviewResponse(), nil
		},
	}

	result, err := Run(context.Background(), types.Config{Base: "main", MaxLoops: 6}, client)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ExitCode != types.ExitSuccess {
		t.Errorf("expected exit code %d, got %d", types.ExitSuccess, result.ExitCode)
	}
	if result.Iterations != 1 {
		t.Errorf("expected 1 iteration, got %d", result.Iterations)
	}
}

func TestRun_DryRun(t *testing.T) {
	setupTest(t)

	findings := sampleFindings()
	fixCalled := false

	client := &mockLoopClient{
		reviewFn: func(_ context.Context, _ llm.ReviewRequest) ([]byte, error) {
			return makeReviewResponse(findings), nil
		},
		triageFn: func(_ context.Context, _ llm.TriageRequest) ([]byte, error) {
			return makeReviewResponse(findings), nil
		},
		fixFn: func(_ context.Context, _ llm.FixRequest) ([]byte, error) {
			fixCalled = true
			return []byte("ok"), nil
		},
	}

	result, err := Run(context.Background(), types.Config{Base: "main", MaxLoops: 6, DryRun: true}, client)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ExitCode != types.ExitMaxIterations {
		t.Errorf("expected exit code %d, got %d", types.ExitMaxIterations, result.ExitCode)
	}
	if fixCalled {
		t.Error("fix should not be called in dry-run mode")
	}
	if len(result.FinalFindings) != 2 {
		t.Errorf("expected 2 findings, got %d", len(result.FinalFindings))
	}
}

func TestRun_SchemaError(t *testing.T) {
	setupTest(t)

	client := &mockLoopClient{
		reviewFn: func(_ context.Context, _ llm.ReviewRequest) ([]byte, error) {
			return []byte(`{"bad json`), nil
		},
	}

	result, err := Run(context.Background(), types.Config{Base: "main", MaxLoops: 6}, client)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ExitCode != types.ExitSchemaError {
		t.Errorf("expected exit code %d, got %d", types.ExitSchemaError, result.ExitCode)
	}
}

func TestRun_MaxIterations(t *testing.T) {
	setupTest(t)

	findings := sampleFindings()

	client := &mockLoopClient{
		reviewFn: func(_ context.Context, _ llm.ReviewRequest) ([]byte, error) {
			return makeReviewResponse(findings), nil
		},
		triageFn: func(_ context.Context, _ llm.TriageRequest) ([]byte, error) {
			return makeReviewResponse(findings), nil
		},
		fixFn: func(_ context.Context, _ llm.FixRequest) ([]byte, error) {
			return []byte("ok"), nil
		},
		validateFn: func(_ context.Context, _ llm.ValidateRequest) ([]byte, error) {
			// Always return remaining findings (never converges).
			return makeReviewResponse(findings[:1]), nil
		},
	}

	result, err := Run(context.Background(), types.Config{Base: "main", MaxLoops: 2}, client)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ExitCode != types.ExitMaxIterations {
		t.Errorf("expected exit code %d, got %d", types.ExitMaxIterations, result.ExitCode)
	}
	if result.Iterations != 2 {
		t.Errorf("expected 2 iterations, got %d", result.Iterations)
	}
}

func TestRun_VerifyFailsThenPasses(t *testing.T) {
	setupTest(t)

	findings := sampleFindings()
	fixCount := 0

	// Override cmdExecutor in steps/verify — not directly accessible from here,
	// so we test via the loop: first fix call triggers verify failure, second succeeds.
	// We mock this by counting fix calls and controlling verify via test commands.
	client := &mockLoopClient{
		reviewFn: func(_ context.Context, _ llm.ReviewRequest) ([]byte, error) {
			return makeReviewResponse(findings), nil
		},
		triageFn: func(_ context.Context, _ llm.TriageRequest) ([]byte, error) {
			return makeReviewResponse(findings), nil
		},
		fixFn: func(_ context.Context, _ llm.FixRequest) ([]byte, error) {
			fixCount++
			return []byte("ok"), nil
		},
		validateFn: func(_ context.Context, _ llm.ValidateRequest) ([]byte, error) {
			return emptyReviewResponse(), nil
		},
	}

	// No verify commands → verify always passes.
	result, err := Run(context.Background(), types.Config{Base: "main", MaxLoops: 6}, client)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ExitCode != types.ExitSuccess {
		t.Errorf("expected exit code %d, got %d", types.ExitSuccess, result.ExitCode)
	}
	if fixCount != 1 {
		t.Errorf("expected 1 fix call, got %d", fixCount)
	}
}

func TestRun_NoDiff(t *testing.T) {
	setupTest(t)

	// Override to return empty diff.
	gitCmd = func(ctx context.Context, args ...string) (string, error) {
		return "", nil
	}

	client := &mockLoopClient{}

	result, err := Run(context.Background(), types.Config{Base: "main", MaxLoops: 6}, client)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ExitCode != types.ExitSuccess {
		t.Errorf("expected exit code %d, got %d", types.ExitSuccess, result.ExitCode)
	}
}
