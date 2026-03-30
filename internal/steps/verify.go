package steps

import (
	"context"
	"os/exec"

	"github.com/knwoop/ccpatrol/internal/types"
)

// VerifyResult holds the outcome of running verification commands.
type VerifyResult struct {
	Passed          bool
	TestsFailed     bool
	TestOutput      string
	LintFailed      bool
	LintOutput      string
	TypecheckFailed bool
	TypecheckOutput string
}

// cmdExecutor is the function used to execute shell commands. Tests can replace it.
var cmdExecutor = defaultCmdExecutor

func defaultCmdExecutor(ctx context.Context, command string) ([]byte, error) {
	return exec.CommandContext(ctx, "sh", "-c", command).CombinedOutput()
}

// Verify runs the configured test, lint, and typecheck commands and returns
// a result summarising which (if any) failed.
func Verify(ctx context.Context, cfg types.Config) (*VerifyResult, error) {
	result := &VerifyResult{}

	if cfg.TestCmd != "" {
		out, err := cmdExecutor(ctx, cfg.TestCmd)
		if err != nil {
			result.TestsFailed = true
			result.TestOutput = string(out)
		}
	}

	if cfg.LintCmd != "" {
		out, err := cmdExecutor(ctx, cfg.LintCmd)
		if err != nil {
			result.LintFailed = true
			result.LintOutput = string(out)
		}
	}

	if cfg.TypecheckCmd != "" {
		out, err := cmdExecutor(ctx, cfg.TypecheckCmd)
		if err != nil {
			result.TypecheckFailed = true
			result.TypecheckOutput = string(out)
		}
	}

	result.Passed = !result.TestsFailed && !result.LintFailed && !result.TypecheckFailed
	return result, nil
}
