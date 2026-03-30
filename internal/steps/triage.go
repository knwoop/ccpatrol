package steps

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/knwoop/ccpatrol/internal/llm"
	"github.com/knwoop/ccpatrol/internal/oscillation"
	"github.com/knwoop/ccpatrol/internal/schema"
	"github.com/knwoop/ccpatrol/internal/state"
	"github.com/knwoop/ccpatrol/internal/types"
)

const maxTriageAttempts = 3

// TriageResult holds the categorised output of the Triage step.
type TriageResult struct {
	Actionable    []types.Finding
	DroppedLow    []types.Finding
	DroppedFP     []types.Finding
	DroppedConf   []types.Finding
	Oscillating   []types.Finding
	NewDirectives []state.Directive
}

// Triage performs a two-phase triage of code review findings.
//
// Phase 1 calls the LLM to re-evaluate findings (potentially changing severity
// or marking false positives) and validates the response against the schema,
// retrying up to 2 times on validation failure.
//
// Phase 2 deterministically filters the LLM-returned findings into actionable,
// dropped, and oscillating buckets.
func Triage(
	ctx context.Context,
	client llm.Client,
	findings []types.Finding,
	st *state.State,
	changedFiles string,
	diffFiles []string,
	prompt string,
	cfg types.Config,
) (*TriageResult, error) {
	// Phase 1: LLM triage with schema validation and retries.
	findingsJSON, err := json.Marshal(findings)
	if err != nil {
		return nil, fmt.Errorf("marshalling findings: %w", err)
	}

	req := llm.TriageRequest{
		Findings:     string(findingsJSON),
		ChangedFiles: changedFiles,
		PromptText:   prompt,
		Verbose:      cfg.Verbose,
	}

	var result *types.ReviewResult
	var lastErr error
	for attempt := range maxTriageAttempts {
		raw, callErr := client.Triage(ctx, req)
		if callErr != nil {
			return nil, fmt.Errorf("llm triage call failed: %w", callErr)
		}

		validated, valErr := schema.Validate(raw)
		if valErr != nil {
			lastErr = fmt.Errorf("attempt %d: schema validation failed: %w", attempt+1, valErr)
			continue
		}

		result = validated
		break
	}
	if result == nil {
		return nil, fmt.Errorf("triage failed after %d attempts: %w", maxTriageAttempts, lastErr)
	}

	// Phase 2: deterministic filtering.
	diffSet := make(map[string]bool, len(diffFiles))
	for _, f := range diffFiles {
		diffSet[f] = true
	}

	// Convert state history to oscillation snapshots.
	history := make([]oscillation.IterationSnapshot, len(st.History))
	for i, rec := range st.History {
		history[i] = oscillation.IterationSnapshot{
			Iteration: rec.Iteration,
			Findings:  rec.FindingsSnapshot,
		}
	}

	tr := &TriageResult{}

	for _, f := range result.Findings {
		switch {
		case f.IsFalsePositive:
			tr.DroppedFP = append(tr.DroppedFP, f)
		case f.Severity == "LOW":
			tr.DroppedLow = append(tr.DroppedLow, f)
		case f.Confidence == "LOW":
			tr.DroppedConf = append(tr.DroppedConf, f)
		case !diffSet[f.File]:
			tr.DroppedLow = append(tr.DroppedLow, f)
		case oscillation.IsOscillating(f, history):
			tr.Oscillating = append(tr.Oscillating, f)
			tr.NewDirectives = append(tr.NewDirectives, state.Directive{
				File:           f.File,
				Lines:          fmt.Sprintf("%d-%d", f.LineStart, f.LineEnd),
				LockedApproach: f.SuggestedFix,
				LockedAtIter:   st.Iteration,
				Reason:         "oscillation detected",
			})
		default:
			tr.Actionable = append(tr.Actionable, f)
		}
	}

	return tr, nil
}
