package cmd

import (
	"fmt"
	"os"

	"github.com/knwoop/ccpatrol/internal/types"
)

const version = "0.1.0"

// Execute dispatches subcommands based on os.Args.
func Execute(args []string) int {
	if len(args) < 2 {
		printUsage()
		return types.ExitConfigError
	}

	switch args[1] {
	case "review":
		return runReview(args[2:])
	case "version":
		fmt.Printf("ccpatrol v%s\n", version)
		return types.ExitSuccess
	case "help", "--help", "-h":
		printUsage()
		return types.ExitSuccess
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n", args[1])
		printUsage()
		return types.ExitConfigError
	}
}

func printUsage() {
	fmt.Fprintf(os.Stderr, `Usage: ccpatrol <command> [flags]

Commands:
  review    Run automated code review loop
  version   Print version
  help      Show this help

Run 'ccpatrol review --help' for review flags.
`)
}
