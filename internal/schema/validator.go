package schema

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/knwoop/ccpatrol/internal/types"
)

var (
	idPattern       = regexp.MustCompile(`^F[0-9]{3}$`)
	validSeverity   = map[string]bool{"CRITICAL": true, "IMPORTANT": true, "LOW": true}
	validCategory   = map[string]bool{"bug": true, "security": true, "performance": true, "logic": true, "type-safety": true, "error-handling": true, "race-condition": true}
	validConfidence = map[string]bool{"HIGH": true, "MEDIUM": true, "LOW": true}
)

// Validate parses raw JSON and validates it against the findings schema rules.
// Returns the parsed ReviewResult or a descriptive error.
func Validate(data []byte) (*types.ReviewResult, error) {
	var result types.ReviewResult
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("invalid JSON: %w", err)
	}

	// Check required top-level fields by re-parsing as map.
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("invalid JSON: %w", err)
	}
	if _, ok := raw["findings"]; !ok {
		return nil, fmt.Errorf("schema validation failed: missing required field \"findings\"")
	}
	if _, ok := raw["summary"]; !ok {
		return nil, fmt.Errorf("schema validation failed: missing required field \"summary\"")
	}

	// Validate each finding.
	for i, f := range result.Findings {
		if errs := validateFinding(f); len(errs) > 0 {
			return nil, fmt.Errorf("schema validation failed: findings[%d]: %s", i, strings.Join(errs, "; "))
		}
	}

	return &result, nil
}

func validateFinding(f types.Finding) []string {
	var errs []string

	if !idPattern.MatchString(f.ID) {
		errs = append(errs, fmt.Sprintf("id %q does not match pattern ^F[0-9]{3}$", f.ID))
	}
	if !validSeverity[f.Severity] {
		errs = append(errs, fmt.Sprintf("severity %q is not one of CRITICAL, IMPORTANT, LOW", f.Severity))
	}
	if !validCategory[f.Category] {
		errs = append(errs, fmt.Sprintf("category %q is not a valid category", f.Category))
	}
	if f.File == "" {
		errs = append(errs, "file is required")
	}
	if f.LineStart < 1 {
		errs = append(errs, "line_start must be >= 1")
	}
	if f.LineEnd < 1 {
		errs = append(errs, "line_end must be >= 1")
	}
	if len(f.Title) > 120 {
		errs = append(errs, "title exceeds 120 characters")
	}
	if f.SuggestedFix == "" {
		errs = append(errs, "suggested_fix must not be empty")
	}
	if !validConfidence[f.Confidence] {
		errs = append(errs, fmt.Sprintf("confidence %q is not one of HIGH, MEDIUM, LOW", f.Confidence))
	}

	return errs
}
