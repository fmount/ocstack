package ocstack

import (
	"fmt"
	"os"
	"strings"
	t "github.com/fmount/ocstack/template"
	"github.com/fmount/ocstack/llm"
)

// CliCommand -
func CliCommand(q string, s *llm.Session) {
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
		// no session, return
		if s == nil {
			ShowWarn(fmt.Sprintf("No session"))
			return
		}
		profile, err := t.LoadProfile(tokens[1])
		if err != nil {
			ShowWarn(fmt.Sprintf("%s\n", err))
			return
		}
		TermHeader(tokens[1])
		s.Profile = profile
		s.SetContext()
	case tq == "help":
		TermHelper("")
		return
	default:
		fmt.Println("Default!")
		return
	}
}
