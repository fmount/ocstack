package ocstack

import (
	"fmt"
	"os"
	"strings"

)

// CliCommand -
func CliCommand(q string) {
	query := strings.ToLower(q)
	switch {
	case query == "exit" || query == "quit":
		fmt.Println("Bye!")
		// TODO: dump sessions if any
		os.Exit(0)
	case query == "read":
		fmt.Println("TODO: Read input from workspace path")
		// TODO:
		// - workspace is a path where we have assets that can be used as input
		// - workspace path can be set via an ENV variable
	case query == "template":
		// TODO:
		// - set the template path that will be used by the agentic mode
	case query == "help":
		Helper()
		return
	default:
		fmt.Println("Default!")
		return
	}
}
