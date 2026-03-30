package prompts

import "embed"

// FS contains the embedded prompt template files.
//
//go:embed *.md
var FS embed.FS
