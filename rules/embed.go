package rules

import "embed"

// FS contains the embedded default rule files.
//
//go:embed *.yaml
var FS embed.FS
