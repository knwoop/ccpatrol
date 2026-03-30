package steps

import (
	"context"
	"fmt"

	"github.com/knwoop/ccpatrol/internal/llm"
	"github.com/knwoop/ccpatrol/internal/schema"
	"github.com/knwoop/ccpatrol/internal/types"
)

const maxReviewAttempts = 3

// Review sends the already-templated prompt to the LLM client, validates the
// response against the findings schema, and retries up to 3 total attempts if
// schema validation fails.
func Review(ctx context.Context, client llm.Client, prompt string, cfg types.Config) (*types.ReviewResult, error) {
	req := llm.ReviewRequest{
		PromptText: prompt,
		Verbose:    cfg.Verbose,
	}

	var lastErr error
	for attempt := range maxReviewAttempts {
		raw, err := client.Review(ctx, req)
		if err != nil {
			return nil, fmt.Errorf("llm review call failed: %w", err)
		}

		result, err := schema.Validate(raw)
		if err != nil {
			lastErr = fmt.Errorf("attempt %d: schema validation failed: %w", attempt+1, err)
			continue
		}

		return result, nil
	}

	return nil, fmt.Errorf("review failed after %d attempts: %w", maxReviewAttempts, lastErr)
}
