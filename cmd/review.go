package cmd

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/knwoop/ccpatrol/internal/llm"
	"github.com/knwoop/ccpatrol/internal/loop"
	"github.com/knwoop/ccpatrol/internal/types"
)

func runReview(args []string) int {
	fs := flag.NewFlagSet("review", flag.ContinueOnError)

	base := fs.String("base", "main", "base branch for diff")
	maxLoops := fs.Int("max-loops", 6, "maximum review iterations")
	staged := fs.Bool("staged", false, "review staged changes only")
	backend := fs.String("backend", "claude", "LLM backend (claude)")
	testCmd := fs.String("test-cmd", "", "test command (e.g. 'go test ./...')")
	lintCmd := fs.String("lint-cmd", "", "lint command (e.g. 'golangci-lint run')")
	typecheckCmd := fs.String("typecheck-cmd", "", "typecheck command")
	jsonOutput := fs.Bool("json", false, "machine-readable JSON output")
	dryRun := fs.Bool("dry-run", false, "review only, no fixes applied")
	verbose := fs.Bool("verbose", false, "show LLM prompts and responses")

	if err := fs.Parse(args); err != nil {
		if err == flag.ErrHelp {
			return types.ExitSuccess
		}
		return types.ExitConfigError
	}

	cfg := types.Config{
		Base:         *base,
		MaxLoops:     *maxLoops,
		Staged:       *staged,
		Backend:      *backend,
		TestCmd:      *testCmd,
		LintCmd:      *lintCmd,
		TypecheckCmd: *typecheckCmd,
		JSON:         *jsonOutput,
		DryRun:       *dryRun,
		Verbose:      *verbose,
	}

	// Select LLM backend.
	var client llm.Client
	switch cfg.Backend {
	case "claude":
		client = llm.NewClaudeClient(&llm.ExecRunner{})
	default:
		fmt.Fprintf(os.Stderr, "unsupported backend: %s\n", cfg.Backend)
		return types.ExitConfigError
	}

	result, err := loop.Run(context.Background(), cfg, client)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		return types.ExitConfigError
	}

	// Output results.
	if cfg.JSON {
		data, _ := json.MarshalIndent(result, "", "  ")
		fmt.Println(string(data))
	} else {
		printHumanResult(result)
	}

	return result.ExitCode
}

func printHumanResult(r *loop.LoopResult) {
	fmt.Println("─────────────────────────────────")
	fmt.Printf("ccpatrol review — %s\n", r.Summary)
	fmt.Printf("iterations: %d\n", r.Iterations)

	if len(r.FinalFindings) > 0 {
		fmt.Printf("\nunresolved findings (%d):\n", len(r.FinalFindings))
		for _, f := range r.FinalFindings {
			fmt.Printf("  [%s] %s %s:%d-%d — %s\n",
				f.Severity, f.ID, f.File, f.LineStart, f.LineEnd, f.Title)
		}
	}

	fmt.Println("─────────────────────────────────")
	fmt.Println("This review covers code quality only. Architectural and product decisions require human review.")
}
