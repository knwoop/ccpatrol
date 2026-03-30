package llm

import "context"

// Client defines the interface for LLM-backed code review operations.
type Client interface {
	Review(ctx context.Context, req ReviewRequest) ([]byte, error)
	Fix(ctx context.Context, req FixRequest) ([]byte, error)
	Triage(ctx context.Context, req TriageRequest) ([]byte, error)
	Validate(ctx context.Context, req ValidateRequest) ([]byte, error)
}

// ReviewRequest contains parameters for the review step.
type ReviewRequest struct {
	Diff       string
	PromptText string
	Verbose    bool
}

// FixRequest contains parameters for the fix step.
type FixRequest struct {
	Findings   string // JSON findings
	Directives string // locked directives text
	PromptText string
	Verbose    bool
}

// TriageRequest contains parameters for the triage step.
type TriageRequest struct {
	Findings     string // JSON findings
	ChangedFiles string // changed file contents
	PromptText   string
	Verbose      bool
}

// ValidateRequest contains parameters for the validate step.
type ValidateRequest struct {
	OriginalFindings string
	PromptText       string
	Verbose          bool
}
