package cmd

import (
	"testing"

	"github.com/knwoop/ccpatrol/internal/types"
)

func TestExecute_Version(t *testing.T) {
	code := Execute([]string{"ccpatrol", "version"})
	if code != types.ExitSuccess {
		t.Errorf("expected exit %d, got %d", types.ExitSuccess, code)
	}
}

func TestExecute_Help(t *testing.T) {
	code := Execute([]string{"ccpatrol", "help"})
	if code != types.ExitSuccess {
		t.Errorf("expected exit %d, got %d", types.ExitSuccess, code)
	}
}

func TestExecute_NoArgs(t *testing.T) {
	code := Execute([]string{"ccpatrol"})
	if code != types.ExitConfigError {
		t.Errorf("expected exit %d, got %d", types.ExitConfigError, code)
	}
}

func TestExecute_UnknownCommand(t *testing.T) {
	code := Execute([]string{"ccpatrol", "unknown"})
	if code != types.ExitConfigError {
		t.Errorf("expected exit %d, got %d", types.ExitConfigError, code)
	}
}

func TestExecute_ReviewUnsupportedBackend(t *testing.T) {
	code := Execute([]string{"ccpatrol", "review", "-backend", "unsupported"})
	if code != types.ExitConfigError {
		t.Errorf("expected exit %d, got %d", types.ExitConfigError, code)
	}
}
