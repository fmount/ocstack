package ocstack

import (
	"fmt"
	"os"
	"strings"
	t "github.com/fmount/ocstack/templates"
)

// CliCommand -
func CliCommand(q string) {
	query := strings.ToLower(q)
	tokens := strings.Split(query, " ")
	tq := tokens[0]
	// tokenize and get the first item. Next items are passed as parameters to
	// the selected case
	switch {
	case tq == "exit" || tq == "quit":
		fmt.Println("Bye!")
		// TODO: dump sessions if any
		os.Exit(0)
	case tq == "read":
		fmt.Println("TODO: Read input from workspace path")
		// TODO:
		// - workspace is a path where we have assets that can be used as input
		// - workspace path can be set via an ENV variable
	case tq == "template":
		if len(tokens) < 2 {
			TermHelper(tq)
			return
		}
		_, err := t.LoadProfile(tokens[1])
		if err != nil {
			ShowWarn(fmt.Sprintf("%s\n", err))
		}
		TermHeader("default")
		// TODO: call llm.api to set the new context
	case tq == "help":
		TermHelper("")
		return
	default:
		fmt.Println("Default!")
		return
	}
}
