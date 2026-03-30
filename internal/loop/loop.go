package loop

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os/exec"
	"strings"
	"text/template"

	"github.com/knwoop/ccpatrol/internal/llm"
	"github.com/knwoop/ccpatrol/internal/state"
	"github.com/knwoop/ccpatrol/internal/steps"
	"github.com/knwoop/ccpatrol/internal/types"
	"github.com/knwoop/ccpatrol/prompts"
)

// LoopResult is the structured outcome of the review loop.
type LoopResult struct {
	ExitCode      int              `json:"exit_code"`
	Iterations    int              `json:"iterations"`
	FinalFindings []types.Finding  `json:"final_findings,omitempty"`
	Summary       string           `json:"summary"`
}

const (
	maxFixAttempts = 3
	stateFile      = ".auto-review-state.json"
)

// gitCmd runs a git command and returns its stdout. Tests can replace this.
var gitCmd = func(ctx context.Context, args ...string) (string, error) {
	out, err := exec.CommandContext(ctx, "git", args...).Output()
	if err != nil {
		return "", fmt.Errorf("git %s: %w", args[0], err)
	}
	return strings.TrimSpace(string(out)), nil
}

// readFile reads a file's contents. Tests can replace this.
var readFile = func(path string) (string, error) {
	out, err := exec.Command("cat", path).Output()
	if err != nil {
		return "", err
	}
	return string(out), nil
}

// Run executes the deterministic review loop.
func Run(ctx context.Context, cfg types.Config, client llm.Client) (*LoopResult, error) {
	// Load or create state.
	st, err := state.Load(stateFile)
	if err != nil {
		return &LoopResult{ExitCode: types.ExitConfigError, Summary: fmt.Sprintf("failed to load state: %v", err)}, nil
	}
	st.MaxIterations = cfg.MaxLoops
	st.BaseBranch = cfg.Base

	// Load prompt templates.
	tmpl, err := loadTemplates()
	if err != nil {
		return &LoopResult{ExitCode: types.ExitConfigError, Summary: fmt.Sprintf("failed to load templates: %v", err)}, nil
	}

	for iteration := 1; iteration <= cfg.MaxLoops; iteration++ {
		slog.Info("starting iteration", "iteration", iteration, "max", cfg.MaxLoops)

		// Compute diff.
		diff, err := computeDiff(ctx, cfg)
		if err != nil {
			return &LoopResult{ExitCode: types.ExitConfigError, Summary: fmt.Sprintf("failed to compute diff: %v", err)}, nil
		}
		if diff == "" {
			return &LoopResult{ExitCode: types.ExitSuccess, Summary: "no changes to review"}, nil
		}

		// === REVIEW ===
		reviewPrompt, err := renderTemplate(tmpl, "review.md", map[string]string{"Diff": diff})
		if err != nil {
			return nil, fmt.Errorf("render review prompt: %w", err)
		}

		reviewResult, err := steps.Review(ctx, client, reviewPrompt, cfg)
		if err != nil {
			return &LoopResult{ExitCode: types.ExitSchemaError, Summary: fmt.Sprintf("review failed: %v", err)}, nil
		}

		if len(reviewResult.Findings) == 0 {
			return &LoopResult{ExitCode: types.ExitSuccess, Iterations: iteration, Summary: "no findings"}, nil
		}

		// === TRIAGE ===
		diffFiles, err := getDiffFiles(ctx, cfg)
		if err != nil {
			return &LoopResult{ExitCode: types.ExitConfigError, Summary: fmt.Sprintf("failed to get diff files: %v", err)}, nil
		}

		changedFilesContent, err := getChangedFilesContent(diffFiles)
		if err != nil {
			return &LoopResult{ExitCode: types.ExitConfigError, Summary: fmt.Sprintf("failed to read changed files: %v", err)}, nil
		}

		findingsJSON, _ := json.MarshalIndent(reviewResult.Findings, "", "  ")
		triagePrompt, err := renderTemplate(tmpl, "triage.md", map[string]string{
			"Findings":     string(findingsJSON),
			"ChangedFiles": changedFilesContent,
		})
		if err != nil {
			return nil, fmt.Errorf("render triage prompt: %w", err)
		}

		triageResult, err := steps.Triage(ctx, client, reviewResult.Findings, st, changedFilesContent, diffFiles, triagePrompt, cfg)
		if err != nil {
			return &LoopResult{ExitCode: types.ExitSchemaError, Summary: fmt.Sprintf("triage failed: %v", err)}, nil
		}

		// Apply new directives.
		st.Directives = append(st.Directives, triageResult.NewDirectives...)

		if len(triageResult.Actionable) == 0 {
			return &LoopResult{
				ExitCode:   types.ExitSuccess,
				Iterations: iteration,
				Summary:    "all findings resolved in triage",
			}, nil
		}

		// === DRY-RUN ===
		if cfg.DryRun {
			return &LoopResult{
				ExitCode:      types.ExitMaxIterations,
				Iterations:    iteration,
				FinalFindings: triageResult.Actionable,
				Summary:       fmt.Sprintf("dry-run: %d actionable findings", len(triageResult.Actionable)),
			}, nil
		}

		// === FIX + VERIFY inner loop ===
		actionableJSON, _ := json.MarshalIndent(triageResult.Actionable, "", "  ")
		directivesText := formatDirectives(st.Directives)

		var verifyResult *steps.VerifyResult
		for fixAttempt := range maxFixAttempts {
			fixPrompt, err := renderTemplate(tmpl, "fix.md", map[string]string{
				"Findings":   string(actionableJSON),
				"Directives": directivesText,
			})
			if err != nil {
				return nil, fmt.Errorf("render fix prompt: %w", err)
			}

			if err := steps.Fix(ctx, client, fixPrompt, cfg); err != nil {
				return &LoopResult{ExitCode: types.ExitMaxIterations, Summary: fmt.Sprintf("fix failed: %v", err)}, nil
			}

			// VERIFY
			verifyResult, err = steps.Verify(ctx, cfg)
			if err != nil {
				return nil, fmt.Errorf("verify error: %w", err)
			}
			if verifyResult.Passed {
				break
			}

			slog.Info("verify failed, retrying fix", "attempt", fixAttempt+1, "max", maxFixAttempts)

			// Feed verify failures back as the "findings" for the next fix attempt.
			actionableJSON = buildVerifyFailureFindings(verifyResult)
		}

		if !verifyResult.Passed {
			return &LoopResult{
				ExitCode:      types.ExitMaxIterations,
				Iterations:    iteration,
				FinalFindings: triageResult.Actionable,
				Summary:       fmt.Sprintf("verify failed after %d fix attempts", maxFixAttempts),
			}, nil
		}

		// === VALIDATE ===
		validatePrompt, err := renderTemplate(tmpl, "validate.md", map[string]string{
			"OriginalFindings": string(actionableJSON),
		})
		if err != nil {
			return nil, fmt.Errorf("render validate prompt: %w", err)
		}

		validateResult, err := steps.ValidateStep(ctx, client, validatePrompt, cfg)
		if err != nil {
			return &LoopResult{ExitCode: types.ExitSchemaError, Summary: fmt.Sprintf("validate failed: %v", err)}, nil
		}

		if len(validateResult.Findings) == 0 {
			return &LoopResult{
				ExitCode:   types.ExitSuccess,
				Iterations: iteration,
				Summary:    fmt.Sprintf("all findings resolved in %d iteration(s)", iteration),
			}, nil
		}

		// === UPDATE STATE ===
		record := buildIterationRecord(iteration, reviewResult, triageResult, verifyResult, validateResult)
		st.AddIteration(record)
		if err := st.Save(stateFile); err != nil {
			slog.Error("failed to save state", "error", err)
		}

		slog.Info("iteration complete", "iteration", iteration, "remaining", len(validateResult.Findings))
	}

	return &LoopResult{
		ExitCode:   types.ExitMaxIterations,
		Iterations: cfg.MaxLoops,
		Summary:    fmt.Sprintf("max iterations (%d) reached", cfg.MaxLoops),
	}, nil
}

func loadTemplates() (*template.Template, error) {
	return template.New("").ParseFS(prompts.FS, "*.md")
}

func renderTemplate(tmpl *template.Template, name string, data any) (string, error) {
	var buf bytes.Buffer
	if err := tmpl.ExecuteTemplate(&buf, name, data); err != nil {
		return "", err
	}
	return buf.String(), nil
}

func computeDiff(ctx context.Context, cfg types.Config) (string, error) {
	if cfg.Staged {
		return gitCmd(ctx, "diff", "--staged")
	}
	return gitCmd(ctx, "diff", cfg.Base+"...HEAD")
}

func getDiffFiles(ctx context.Context, cfg types.Config) ([]string, error) {
	var out string
	var err error
	if cfg.Staged {
		out, err = gitCmd(ctx, "diff", "--staged", "--name-only")
	} else {
		out, err = gitCmd(ctx, "diff", cfg.Base+"...HEAD", "--name-only")
	}
	if err != nil {
		return nil, err
	}
	if out == "" {
		return nil, nil
	}
	return strings.Split(out, "\n"), nil
}

func getChangedFilesContent(files []string) (string, error) {
	var b strings.Builder
	for _, f := range files {
		content, err := readFile(f)
		if err != nil {
			slog.Warn("could not read file", "file", f, "error", err)
			continue
		}
		fmt.Fprintf(&b, "=== %s ===\n%s\n\n", f, content)
	}
	return b.String(), nil
}

func formatDirectives(directives []state.Directive) string {
	if len(directives) == 0 {
		return ""
	}
	var b strings.Builder
	for _, d := range directives {
		fmt.Fprintf(&b, "- %s:%s — %s (locked at iteration %d: %s)\n",
			d.File, d.Lines, d.LockedApproach, d.LockedAtIter, d.Reason)
	}
	return b.String()
}

func buildVerifyFailureFindings(v *steps.VerifyResult) []byte {
	var findings []types.Finding
	id := 900

	if v.TestsFailed {
		findings = append(findings, types.Finding{
			ID:           fmt.Sprintf("F%03d", id),
			Severity:     "CRITICAL",
			Category:     "bug",
			File:         "test output",
			LineStart:    1,
			LineEnd:      1,
			Title:        "Tests failing after fix",
			Explanation:  v.TestOutput,
			SuggestedFix: "Fix the test failures shown above",
			Confidence:   "HIGH",
		})
		id++
	}
	if v.LintFailed {
		findings = append(findings, types.Finding{
			ID:           fmt.Sprintf("F%03d", id),
			Severity:     "IMPORTANT",
			Category:     "bug",
			File:         "lint output",
			LineStart:    1,
			LineEnd:      1,
			Title:        "Linter errors after fix",
			Explanation:  v.LintOutput,
			SuggestedFix: "Fix the lint errors shown above",
			Confidence:   "HIGH",
		})
		id++
	}
	if v.TypecheckFailed {
		findings = append(findings, types.Finding{
			ID:           fmt.Sprintf("F%03d", id),
			Severity:     "CRITICAL",
			Category:     "type-safety",
			File:         "typecheck output",
			LineStart:    1,
			LineEnd:      1,
			Title:        "Type errors after fix",
			Explanation:  v.TypecheckOutput,
			SuggestedFix: "Fix the type errors shown above",
			Confidence:   "HIGH",
		})
	}

	data, _ := json.MarshalIndent(findings, "", "  ")
	return data
}

func buildIterationRecord(
	iteration int,
	review *types.ReviewResult,
	triage *steps.TriageResult,
	verify *steps.VerifyResult,
	validate *types.ReviewResult,
) state.IterationRecord {
	rec := state.IterationRecord{
		Iteration:        iteration,
		VerifyPassed:     verify.Passed,
		FindingsSnapshot: review.Findings,
	}

	for _, f := range review.Findings {
		rec.FindingsIDs = append(rec.FindingsIDs, f.ID)
	}

	droppedSet := make(map[string]bool)
	for _, f := range triage.DroppedLow {
		droppedSet[f.ID] = true
	}
	for _, f := range triage.DroppedFP {
		droppedSet[f.ID] = true
	}
	for _, f := range triage.DroppedConf {
		droppedSet[f.ID] = true
	}
	for _, f := range triage.Oscillating {
		droppedSet[f.ID] = true
	}
	for id := range droppedSet {
		rec.DroppedIDs = append(rec.DroppedIDs, id)
	}

	remainingSet := make(map[string]bool)
	for _, f := range validate.Findings {
		remainingSet[f.ID] = true
	}
	for _, f := range triage.Actionable {
		if remainingSet[f.ID] {
			rec.RemainingIDs = append(rec.RemainingIDs, f.ID)
		} else {
			rec.FixedIDs = append(rec.FixedIDs, f.ID)
		}
	}

	if !verify.Passed {
		if verify.TestsFailed {
			rec.VerifyFailures = append(rec.VerifyFailures, "tests: "+verify.TestOutput)
		}
		if verify.LintFailed {
			rec.VerifyFailures = append(rec.VerifyFailures, "lint: "+verify.LintOutput)
		}
		if verify.TypecheckFailed {
			rec.VerifyFailures = append(rec.VerifyFailures, "typecheck: "+verify.TypecheckOutput)
		}
	}

	return rec
}
