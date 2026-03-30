package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os/exec"
)

// CommandRunner abstracts command execution for testability.
type CommandRunner interface {
	Run(ctx context.Context, name string, args ...string) ([]byte, error)
}

// ExecRunner implements CommandRunner using os/exec.
type ExecRunner struct{}

// Run executes the named program with the given arguments and returns its
// combined stdout output. If the command exits with a non-zero status, the
// returned error is of type *exec.ExitError.
func (r *ExecRunner) Run(ctx context.Context, name string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	return cmd.Output()
}

// ClaudeClient implements Client by shelling out to the claude CLI.
type ClaudeClient struct {
	runner CommandRunner
}

// NewClaudeClient returns a new ClaudeClient backed by the given CommandRunner.
func NewClaudeClient(runner CommandRunner) *ClaudeClient {
	return &ClaudeClient{runner: runner}
}

// claudeMessage represents a single message in the claude CLI JSON output.
type claudeMessage struct {
	Type   string `json:"type"`
	Result string `json:"result"`
}

// parseClaudeOutput extracts the result text from the claude CLI JSON envelope.
// The CLI returns a JSON array; the last element with type "result" holds the
// actual LLM output in its "result" field.
func parseClaudeOutput(raw []byte) ([]byte, error) {
	var messages []claudeMessage
	if err := json.Unmarshal(raw, &messages); err != nil {
		return nil, fmt.Errorf("parsing claude JSON output: %w", err)
	}

	for i := len(messages) - 1; i >= 0; i-- {
		if messages[i].Type == "result" {
			return []byte(messages[i].Result), nil
		}
	}

	return nil, fmt.Errorf("no result message found in claude output")
}

// runClaude executes the claude CLI with the given prompt and options, then
// parses and returns the result text.
func (c *ClaudeClient) runClaude(ctx context.Context, prompt string, maxTurns int, verbose bool) ([]byte, error) {
	args := []string{"-p", prompt, "--output-format", "json"}
	if maxTurns > 0 {
		args = append(args, "--max-turns", fmt.Sprintf("%d", maxTurns))
	}

	if verbose {
		slog.Info("running claude command", "args", args)
	}

	out, err := c.runner.Run(ctx, "claude", args...)
	if err != nil {
		return nil, fmt.Errorf("claude command failed: %w", err)
	}

	if verbose {
		slog.Info("claude response", "raw", string(out))
	}

	return parseClaudeOutput(out)
}

// Review runs the review step via the claude CLI.
func (c *ClaudeClient) Review(ctx context.Context, req ReviewRequest) ([]byte, error) {
	return c.runClaude(ctx, req.PromptText, 1, req.Verbose)
}

// Fix runs the fix step via the claude CLI. No --max-turns limit is set
// because the fix may require multiple tool calls.
func (c *ClaudeClient) Fix(ctx context.Context, req FixRequest) ([]byte, error) {
	return c.runClaude(ctx, req.PromptText, 0, req.Verbose)
}

// Triage runs the triage step via the claude CLI.
func (c *ClaudeClient) Triage(ctx context.Context, req TriageRequest) ([]byte, error) {
	return c.runClaude(ctx, req.PromptText, 1, req.Verbose)
}

// Validate runs the validate step via the claude CLI.
func (c *ClaudeClient) Validate(ctx context.Context, req ValidateRequest) ([]byte, error) {
	return c.runClaude(ctx, req.PromptText, 1, req.Verbose)
}
