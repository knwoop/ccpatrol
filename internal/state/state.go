package state

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"

	"github.com/knwoop/ccpatrol/internal/types"
)

// State holds the full patrol run state across iterations.
type State struct {
	Iteration     int               `json:"iteration"`
	MaxIterations int               `json:"max_iterations"`
	BaseBranch    string            `json:"base_branch"`
	VerifyConfig  VerifyConfig      `json:"verify_config"`
	Directives    []Directive       `json:"directives"`
	History       []IterationRecord `json:"history"`
}

// VerifyConfig holds the commands used to verify fixes.
type VerifyConfig struct {
	TestCmd      string `json:"test_cmd"`
	LintCmd      string `json:"lint_cmd"`
	TypecheckCmd string `json:"typecheck_cmd"`
}

// Directive records a locked approach for a specific file location.
type Directive struct {
	File           string `json:"file"`
	Lines          string `json:"lines"`
	LockedApproach string `json:"locked_approach"`
	LockedAtIter   int    `json:"locked_at_iteration"`
	Reason         string `json:"reason"`
}

// IterationRecord captures the outcome of a single iteration.
type IterationRecord struct {
	Iteration        int             `json:"iteration"`
	FindingsIDs      []string        `json:"findings_ids"`
	FixedIDs         []string        `json:"fixed_ids"`
	RemainingIDs     []string        `json:"remaining_ids"`
	DroppedIDs       []string        `json:"dropped_ids"`
	VerifyPassed     bool            `json:"verify_passed"`
	VerifyFailures   []string        `json:"verify_failures"`
	FindingsSnapshot []types.Finding `json:"findings_snapshot"`
}

// Load reads state from the given path. If the file does not exist, it returns
// a fresh empty State (this is not an error).
func Load(path string) (*State, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return &State{}, nil
		}
		return nil, err
	}

	var s State
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, err
	}
	return &s, nil
}

// Save writes the state to the given path atomically. It writes to a temporary
// file in the same directory and then renames it into place so that readers
// never see a partially-written file.
func (s *State) Save(path string) error {
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}

	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, "ccpatrol-state-*.tmp")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()

	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		os.Remove(tmpName)
		return err
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpName)
		return err
	}

	if err := os.Rename(tmpName, path); err != nil {
		os.Remove(tmpName)
		return err
	}
	return nil
}

// AddIteration appends the given record to History and increments the Iteration
// counter.
func (s *State) AddIteration(record IterationRecord) {
	s.History = append(s.History, record)
	s.Iteration++
}
