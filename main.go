package main

import (
	"fmt"
	"os"

	"github.com/knwoop/ccpatrol/cmd"
)

func main() {
	code := cmd.Execute(os.Args)
	if code != 0 {
		fmt.Fprintf(os.Stderr, "exit code: %d\n", code)
	}
	os.Exit(code)
}
