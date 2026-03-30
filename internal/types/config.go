package types

// Config holds all CLI configuration parsed from flags.
type Config struct {
	Base         string
	MaxLoops     int
	Staged       bool
	Backend      string
	TestCmd      string
	LintCmd      string
	TypecheckCmd string
	JSON         bool
	DryRun       bool
	Verbose      bool
}
