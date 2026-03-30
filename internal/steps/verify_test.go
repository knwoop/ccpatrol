package steps

import (
	"context"
	"fmt"
	"testing"

	"github.com/knwoop/ccpatrol/internal/types"
)

// fakeCmdExecutor returns a cmdExecutor that uses the responses map to
// determine output and error for each command string. Commands not in the
// map succeed with empty output.
func fakeCmdExecutor(responses map[string]struct {
	output []byte
	err    error
}) func(context.Context, string) ([]byte, error) {
	return func(_ context.Context, command string) ([]byte, error) {
		if r, ok := responses[command]; ok {
			return r.output, r.err
		}
		return nil, nil
	}
}

func TestVerify_AllCommandsPass(t *testing.T) {
	orig := cmdExecutor
	defer func() { cmdExecutor = orig }()

	cmdExecutor = fakeCmdExecutor(map[string]struct {
		output []byte
		err    error
	}{
		"go test ./...":       {output: []byte("ok"), err: nil},
		"golangci-lint run":   {output: []byte("ok"), err: nil},
		"go vet ./...":        {output: []byte("ok"), err: nil},
	})

	cfg := types.Config{
		TestCmd:      "go test ./...",
		LintCmd:      "golangci-lint run",
		TypecheckCmd: "go vet ./...",
	}

	result, err := Verify(context.Background(), cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Passed {
		t.Fatal("expected Passed to be true")
	}
	if result.TestsFailed {
		t.Fatal("expected TestsFailed to be false")
	}
	if result.LintFailed {
		t.Fatal("expected LintFailed to be false")
	}
	if result.TypecheckFailed {
		t.Fatal("expected TypecheckFailed to be false")
	}
}

func TestVerify_TestCommandFails(t *testing.T) {
	orig := cmdExecutor
	defer func() { cmdExecutor = orig }()

	cmdExecutor = fakeCmdExecutor(map[string]struct {
		output []byte
		err    error
	}{
		"go test ./...": {output: []byte("FAIL: TestFoo"), err: fmt.Errorf("exit status 1")},
	})

	cfg := types.Config{
		TestCmd: "go test ./...",
	}

	result, err := Verify(context.Background(), cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Passed {
		t.Fatal("expected Passed to be false")
	}
	if !result.TestsFailed {
		t.Fatal("expected TestsFailed to be true")
	}
	if result.TestOutput != "FAIL: TestFoo" {
		t.Fatalf("unexpected TestOutput: %q", result.TestOutput)
	}
}

func TestVerify_LintCommandFails(t *testing.T) {
	orig := cmdExecutor
	defer func() { cmdExecutor = orig }()

	cmdExecutor = fakeCmdExecutor(map[string]struct {
		output []byte
		err    error
	}{
		"golangci-lint run": {output: []byte("lint error: unused variable"), err: fmt.Errorf("exit status 1")},
	})

	cfg := types.Config{
		LintCmd: "golangci-lint run",
	}

	result, err := Verify(context.Background(), cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Passed {
		t.Fatal("expected Passed to be false")
	}
	if !result.LintFailed {
		t.Fatal("expected LintFailed to be true")
	}
	if result.LintOutput != "lint error: unused variable" {
		t.Fatalf("unexpected LintOutput: %q", result.LintOutput)
	}
}

func TestVerify_MultipleFailures(t *testing.T) {
	orig := cmdExecutor
	defer func() { cmdExecutor = orig }()

	cmdExecutor = fakeCmdExecutor(map[string]struct {
		output []byte
		err    error
	}{
		"go test ./...":     {output: []byte("FAIL"), err: fmt.Errorf("exit status 1")},
		"golangci-lint run": {output: []byte("lint error"), err: fmt.Errorf("exit status 1")},
		"go vet ./...":      {output: []byte("vet error"), err: fmt.Errorf("exit status 2")},
	})

	cfg := types.Config{
		TestCmd:      "go test ./...",
		LintCmd:      "golangci-lint run",
		TypecheckCmd: "go vet ./...",
	}

	result, err := Verify(context.Background(), cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Passed {
		t.Fatal("expected Passed to be false")
	}
	if !result.TestsFailed {
		t.Fatal("expected TestsFailed to be true")
	}
	if !result.LintFailed {
		t.Fatal("expected LintFailed to be true")
	}
	if !result.TypecheckFailed {
		t.Fatal("expected TypecheckFailed to be true")
	}
	if result.TestOutput != "FAIL" {
		t.Fatalf("unexpected TestOutput: %q", result.TestOutput)
	}
	if result.LintOutput != "lint error" {
		t.Fatalf("unexpected LintOutput: %q", result.LintOutput)
	}
	if result.TypecheckOutput != "vet error" {
		t.Fatalf("unexpected TypecheckOutput: %q", result.TypecheckOutput)
	}
}

func TestVerify_NoCommandsConfigured(t *testing.T) {
	orig := cmdExecutor
	defer func() { cmdExecutor = orig }()

	cmdExecutor = fakeCmdExecutor(nil)

	cfg := types.Config{}

	result, err := Verify(context.Background(), cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Passed {
		t.Fatal("expected Passed to be true when no commands configured")
	}
	if result.TestsFailed || result.LintFailed || result.TypecheckFailed {
		t.Fatal("expected no failures when no commands configured")
	}
}
