package rules

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/goccy/go-yaml"

	rulesdata "github.com/knwoop/ccpatrol/rules"
)

// RuleFile represents a parsed rules YAML file.
type RuleFile struct {
	Language    string              `yaml:"language"`
	Description string             `yaml:"description"`
	References  []string           `yaml:"references,omitempty"`
	Rules       map[string][]string `yaml:"rules"`
}

// Load loads and merges rules: embedded defaults + language-specific + project overrides.
// Language is auto-detected from the diff file extensions if not specified.
func Load(diffFiles []string, projectDir string) (string, error) {
	// 1. Load embedded default rules.
	defaultRules, err := loadEmbedded("default.yaml")
	if err != nil {
		return "", fmt.Errorf("loading default rules: %w", err)
	}

	// 2. Detect language and load language-specific rules.
	lang := detectLanguage(diffFiles)
	var langRules *RuleFile
	if lang != "" {
		langRules, err = loadEmbedded(lang + ".yaml")
		if err != nil {
			// Language file doesn't exist — not an error, just skip.
			langRules = nil
		}
	}

	// 3. Load project-level overrides if present.
	var projectRules *RuleFile
	projectPath := filepath.Join(projectDir, ".ccpatrol", "rules.yaml")
	if data, err := os.ReadFile(projectPath); err == nil {
		projectRules = &RuleFile{}
		if err := yaml.Unmarshal(data, projectRules); err != nil {
			return "", fmt.Errorf("parsing %s: %w", projectPath, err)
		}
	}

	// 4. Render merged rules as text for prompt injection.
	return render(defaultRules, langRules, projectRules), nil
}

func loadEmbedded(name string) (*RuleFile, error) {
	data, err := fs.ReadFile(rulesdata.FS, name)
	if err != nil {
		return nil, err
	}
	var rf RuleFile
	if err := yaml.Unmarshal(data, &rf); err != nil {
		return nil, err
	}
	return &rf, nil
}

// detectLanguage returns the primary language based on file extensions in the diff.
func detectLanguage(files []string) string {
	counts := make(map[string]int)
	for _, f := range files {
		ext := strings.ToLower(filepath.Ext(f))
		switch ext {
		case ".go":
			counts["go"]++
		case ".ts", ".tsx":
			counts["typescript"]++
		case ".js", ".jsx":
			counts["javascript"]++
		case ".py":
			counts["python"]++
		case ".rs":
			counts["rust"]++
		case ".java":
			counts["java"]++
		case ".rb":
			counts["ruby"]++
		}
	}

	best := ""
	bestCount := 0
	for lang, c := range counts {
		if c > bestCount {
			best = lang
			bestCount = c
		}
	}
	return best
}

func render(defaultRules, langRules, projectRules *RuleFile) string {
	var b strings.Builder

	// Default rules.
	if defaultRules != nil {
		renderRuleFile(&b, defaultRules)
	}

	// Language-specific rules.
	if langRules != nil {
		fmt.Fprintf(&b, "\n## %s\n\n", langRules.Description)
		for _, ref := range langRules.References {
			fmt.Fprintf(&b, "Reference: %s\n", ref)
		}
		if len(langRules.References) > 0 {
			b.WriteString("\n")
		}
		renderRules(&b, langRules.Rules)
	}

	// Project overrides.
	if projectRules != nil {
		fmt.Fprintf(&b, "\n## Project-specific rules\n\n")
		if projectRules.Description != "" {
			fmt.Fprintf(&b, "%s\n\n", projectRules.Description)
		}
		for _, ref := range projectRules.References {
			fmt.Fprintf(&b, "Reference: %s\n", ref)
		}
		if len(projectRules.References) > 0 {
			b.WriteString("\n")
		}
		renderRules(&b, projectRules.Rules)
	}

	return b.String()
}

func renderRuleFile(b *strings.Builder, rf *RuleFile) {
	fmt.Fprintf(b, "## %s\n\n", rf.Description)
	renderRules(b, rf.Rules)
}

func renderRules(b *strings.Builder, rules map[string][]string) {
	for category, items := range rules {
		fmt.Fprintf(b, "### %s\n", category)
		for _, item := range items {
			fmt.Fprintf(b, "- %s\n", item)
		}
		b.WriteString("\n")
	}
}
