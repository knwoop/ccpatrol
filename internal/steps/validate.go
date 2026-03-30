package steps

import (
	"context"
	"fmt"

	"github.com/knwoop/ccpatrol/internal/llm"
	"github.com/knwoop/ccpatrol/internal/schema"
	"github.com/knwoop/ccpatrol/internal/types"
)

const maxValidateAttempts = 3

// ValidateStep sends the already-templated prompt to the LLM client's Validate
// method, validates the response against the findings schema, and retries up to
// 3 total attempts if schema validation fails.
func ValidateStep(ctx context.Context, client llm.Client, prompt string, cfg types.Config) (*types.ReviewResult, error) {
	req := llm.ValidateRequest{
		PromptText: prompt,
		Verbose:    cfg.Verbose,
	}

	var lastErr error
	for attempt := range maxValidateAttempts {
		raw, err := client.Validate(ctx, req)
		if err != nil {
			return nil, fmt.Errorf("llm validate call failed: %w", err)
		}

		result, err := schema.Validate(raw)
		if err != nil {
			lastErr = fmt.Errorf("attempt %d: schema validation failed: %w", attempt+1, err)
			continue
		}

		return result, nil
	}

	return nil, fmt.Errorf("validate failed after %d attempts: %w", maxValidateAttempts, lastErr)
}
