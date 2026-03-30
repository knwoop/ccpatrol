package llm

import (
	"context"
	"fmt"
	"strings"
	"testing"
)

// mockRunner records calls and returns preconfigured responses.
type mockRunner struct {
	calls []mockCall
	out   []byte
	err   error
}

type mockCall struct {
	name string
	args []string
}

func (m *mockRunner) Run(_ context.Context, name string, args ...string) ([]byte, error) {
	m.calls = append(m.calls, mockCall{name: name, args: args})
	return m.out, m.err
}

// validClaudeResponse builds a well-formed claude CLI JSON envelope.
func validClaudeResponse(resultText string) []byte {
	// Escape the result text for embedding in JSON.
	escaped := strings.ReplaceAll(resultText, `\`, `\\`)
	escaped = strings.ReplaceAll(escaped, `"`, `\"`)
	escaped = strings.ReplaceAll(escaped, "\n", `\n`)
	return fmt.Appendf(nil,
		`[{"type":"result","subtype":"success","result":"%s","session_id":"test-session","cost_usd":0.01}]`,
		escaped,
	)
}

func TestReviewArgs(t *testing.T) {
	m := &mockRunner{out: validClaudeResponse(`{"findings":[]}`)}
	client := NewClaudeClient(m)

	_, err := client.Review(context.Background(), ReviewRequest{
		Diff:       "diff content",
		PromptText: "review this diff",
		Verbose:    false,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(m.calls) != 1 {
		t.Fatalf("expected 1 call, got %d", len(m.calls))
	}
	call := m.calls[0]

	if call.name != "claude" {
		t.Errorf("expected command 'claude', got %q", call.name)
	}

	args := strings.Join(call.args, " ")

	if !strings.Contains(args, "-p") {
		t.Error("expected -p flag in args")
	}
	if !strings.Contains(args, "--output-format json") {
		t.Error("expected --output-format json in args")
	}
	if !strings.Contains(args, "--max-turns 1") {
		t.Error("expected --max-turns 1 in args for Review")
	}
}

func TestFixArgsNoMaxTurns(t *testing.T) {
	m := &mockRunner{out: validClaudeResponse(`{"findings":[]}`)}
	client := NewClaudeClient(m)

	_, err := client.Fix(context.Background(), FixRequest{
		Findings:   `{"findings":[]}`,
		Directives: "no changes",
		PromptText: "fix these findings",
		Verbose:    false,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(m.calls) != 1 {
		t.Fatalf("expected 1 call, got %d", len(m.calls))
	}
	call := m.calls[0]
	args := strings.Join(call.args, " ")

	if strings.Contains(args, "--max-turns") {
		t.Error("Fix should NOT include --max-turns flag")
	}
	if !strings.Contains(args, "--output-format json") {
		t.Error("expected --output-format json in args")
	}
}

func TestTriageArgs(t *testing.T) {
	m := &mockRunner{out: validClaudeResponse(`{"findings":[]}`)}
	client := NewClaudeClient(m)

	_, err := client.Triage(context.Background(), TriageRequest{
		Findings:     `{"findings":[]}`,
		ChangedFiles: "file contents",
		PromptText:   "triage these findings",
		Verbose:      false,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	call := m.calls[0]
	args := strings.Join(call.args, " ")
	if !strings.Contains(args, "--max-turns 1") {
		t.Error("expected --max-turns 1 in args for Triage")
	}
}

func TestValidateArgs(t *testing.T) {
	m := &mockRunner{out: validClaudeResponse(`{"findings":[]}`)}
	client := NewClaudeClient(m)

	_, err := client.Validate(context.Background(), ValidateRequest{
		OriginalFindings: `{"findings":[]}`,
		PromptText:       "validate these findings",
		Verbose:          false,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	call := m.calls[0]
	args := strings.Join(call.args, " ")
	if !strings.Contains(args, "--max-turns 1") {
		t.Error("expected --max-turns 1 in args for Validate")
	}
}

func TestParseClaudeOutputSuccess(t *testing.T) {
	findingsJSON := `{"findings":[{"id":"F001","severity":"CRITICAL"}],"summary":{"total":1}}`
	raw := validClaudeResponse(findingsJSON)

	result, err := parseClaudeOutput(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if string(result) != findingsJSON {
		t.Errorf("parsed result mismatch:\n  got:  %s\n  want: %s", string(result), findingsJSON)
	}
}

func TestParseClaudeOutputMultipleMessages(t *testing.T) {
	raw := []byte(`[
		{"type":"assistant","result":""},
		{"type":"result","subtype":"success","result":"the real output","session_id":"s","cost_usd":0.01}
	]`)

	result, err := parseClaudeOutput(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if string(result) != "the real output" {
		t.Errorf("expected 'the real output', got %q", string(result))
	}
}

func TestErrorOnCommandFailure(t *testing.T) {
	m := &mockRunner{
		out: nil,
		err: fmt.Errorf("exit status 1"),
	}
	client := NewClaudeClient(m)

	_, err := client.Review(context.Background(), ReviewRequest{
		PromptText: "review",
	})
	if err == nil {
		t.Fatal("expected error for non-zero exit code")
	}
	if !strings.Contains(err.Error(), "claude command failed") {
		t.Errorf("expected 'claude command failed' in error, got: %v", err)
	}
}

func TestErrorOnMalformedJSON(t *testing.T) {
	m := &mockRunner{out: []byte(`not json at all`)}
	client := NewClaudeClient(m)

	_, err := client.Review(context.Background(), ReviewRequest{
		PromptText: "review",
	})
	if err == nil {
		t.Fatal("expected error for malformed JSON")
	}
	if !strings.Contains(err.Error(), "parsing claude JSON output") {
		t.Errorf("expected 'parsing claude JSON output' in error, got: %v", err)
	}
}

func TestErrorOnNoResultMessage(t *testing.T) {
	raw := []byte(`[{"type":"assistant","result":"not a result type"}]`)
	m := &mockRunner{out: raw}
	client := NewClaudeClient(m)

	_, err := client.Review(context.Background(), ReviewRequest{
		PromptText: "review",
	})
	if err == nil {
		t.Fatal("expected error when no result message found")
	}
	if !strings.Contains(err.Error(), "no result message found") {
		t.Errorf("expected 'no result message found' in error, got: %v", err)
	}
}

func TestClientInterfaceCompliance(t *testing.T) {
	// Compile-time check that ClaudeClient implements Client.
	var _ Client = (*ClaudeClient)(nil)
}
