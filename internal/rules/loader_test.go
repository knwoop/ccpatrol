package rules

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDetectLanguage_Go(t *testing.T) {
	files := []string{"main.go", "handler.go", "README.md"}
	if got := detectLanguage(files); got != "go" {
		t.Errorf("expected go, got %s", got)
	}
}

func TestDetectLanguage_Mixed(t *testing.T) {
	files := []string{"main.go", "app.ts", "app.tsx", "util.ts"}
	if got := detectLanguage(files); got != "typescript" {
		t.Errorf("expected typescript, got %s", got)
	}
}

func TestDetectLanguage_NoMatch(t *testing.T) {
	files := []string{"README.md", "Makefile"}
	if got := detectLanguage(files); got != "" {
		t.Errorf("expected empty, got %s", got)
	}
}

func TestLoad_DefaultAndGo(t *testing.T) {
	goFiles := []string{"main.go", "handler.go"}
	result, err := Load(goFiles, "/nonexistent")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should contain default rules.
	if !strings.Contains(result, "Language-agnostic") {
		t.Error("expected default rules in output")
	}

	// Should contain Go rules.
	if !strings.Contains(result, "Effective Go") {
		t.Error("expected Go rules in output")
	}
	if !strings.Contains(result, "Uber Go Style Guide") {
		t.Error("expected Uber reference in output")
	}
}

func TestLoad_DefaultOnly(t *testing.T) {
	mdFiles := []string{"README.md"}
	result, err := Load(mdFiles, "/nonexistent")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(result, "Language-agnostic") {
		t.Error("expected default rules")
	}
	if strings.Contains(result, "Effective Go") {
		t.Error("should not contain Go-specific rules")
	}
}

func TestLoad_ProjectOverride(t *testing.T) {
	tmpDir := t.TempDir()
	ccpatrolDir := filepath.Join(tmpDir, ".ccpatrol")
	os.MkdirAll(ccpatrolDir, 0o755)

	projectRules := `language: custom
description: Our team rules
rules:
  style:
    - "Always use structured logging"
    - "No println in production code"
`
	os.WriteFile(filepath.Join(ccpatrolDir, "rules.yaml"), []byte(projectRules), 0o644)

	result, err := Load([]string{"main.go"}, tmpDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(result, "Our team rules") {
		t.Error("expected project rules in output")
	}
	if !strings.Contains(result, "Always use structured logging") {
		t.Error("expected project rule content")
	}
	// Should still have defaults and Go rules.
	if !strings.Contains(result, "Language-agnostic") {
		t.Error("expected default rules")
	}
	if !strings.Contains(result, "Effective Go") {
		t.Error("expected Go rules")
	}
}
