package steps

import (
	"context"
	"fmt"

	"github.com/knwoop/ccpatrol/internal/llm"
	"github.com/knwoop/ccpatrol/internal/types"
)

// Fix sends the already-templated prompt to the LLM client which applies
// edits to files on disk. No schema validation is needed on the output.
func Fix(ctx context.Context, client llm.Client, prompt string, cfg types.Config) error {
	req := llm.FixRequest{
		PromptText: prompt,
		Verbose:    cfg.Verbose,
	}

	_, err := client.Fix(ctx, req)
	if err != nil {
		return fmt.Errorf("llm fix call failed: %w", err)
	}

	return nil
}
