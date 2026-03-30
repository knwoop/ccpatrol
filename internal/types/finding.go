package types

// Finding represents a single code review finding from the LLM.
type Finding struct {
	ID              string `json:"id"`
	Severity        string `json:"severity"`
	Category        string `json:"category"`
	File            string `json:"file"`
	LineStart       int    `json:"line_start"`
	LineEnd         int    `json:"line_end"`
	Title           string `json:"title"`
	Explanation     string `json:"explanation"`
	SuggestedFix    string `json:"suggested_fix"`
	Confidence      string `json:"confidence"`
	IsFalsePositive bool   `json:"is_false_positive,omitempty"`
}

// ReviewResult is the structured output from REVIEW, TRIAGE LLM, and VALIDATE steps.
type ReviewResult struct {
	Findings []Finding     `json:"findings"`
	Summary  ResultSummary `json:"summary"`
}

// ResultSummary counts findings by severity.
type ResultSummary struct {
	Total    int `json:"total"`
	Critical int `json:"critical"`
	Important int `json:"important"`
	Low      int `json:"low"`
}
