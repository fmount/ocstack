package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/fmount/ocstack/llm"
	"github.com/fmount/ocstack/pkg/ocstack"
	t "github.com/fmount/ocstack/template"
	tools "github.com/fmount/ocstack/tools"
)

const (
	DEBUG = false // switch to true to print additional information
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
			ocstack.TermHelper(tq)
			return
		}
		// no session, return
		if s == nil {
			ocstack.ShowWarn("No session")
			return
		}
		profile, err := t.LoadProfile(tokens[1])
		if err != nil {
			ocstack.ShowWarn(fmt.Sprintf("%s\n", err))
			return
		}
		ocstack.TermHeader(tokens[1])
		s.Profile = profile
		s.UpdateContext()
	case tq == "namespace":
		// set or update namespace
		s.SetConfig(ocstack.NAMESPACE, tokens[1])
	case tq == "config":
		// show config options
		s.ShowConfig()
	case tq == "help":
		ocstack.TermHelper("")
		return
	default:
		fmt.Println("Default!")
		return
	}
}

func main() {

	// Validate ocstack input required to access Tools
	tools.ExitOnErrors()

	ctx := context.Background()

	client, err := llm.GetProvider(llm.OLLAMAPROVIDER)
	if err != nil {
		panic(err)
	}

	h := llm.History{}
	b, err := tools.RegisterTools()
	if err != nil {
		panic(err)
	}

	profile, err := t.LoadProfile("default")
	if err != nil {
		ocstack.ShowWarn(fmt.Sprintf("%s\n", err))
	}

	config := tools.LoadDefaultConfig()

	// Create a new session for the current execution before entering the
	// loop
	s, _ := llm.NewSession(
		llm.QWEN,
		profile,
		h,
		b,
		DEBUG,
		config,
	)

	// pass the loaded profile
	ocstack.TermHeader("default")

	for {
		fmt.Printf("Q :> ")
		// Read input
		reader := bufio.NewReader(os.Stdin)
		input, err := reader.ReadString('\n')
		if err != nil {
			log.Fatal(err)
		}

		// no input provided, go back to the beginning
		if len(strings.TrimSuffix(input, "\n")) == 0 {
			continue
		}

		// process potential commands
		if len(input) > 0 && strings.HasPrefix(input, "/") {
			// Trim any whitespace from the input
			q := strings.TrimSpace(input)
			CliCommand(strings.TrimPrefix(q, "/"), s)
			continue
		}

		// propagate the request to the LLM
		err = client.GenerateChat(
			ctx,
			input,
			s,
		)
		if s.Debug {
			fmt.Printf("[HISTORY]:\n")
			fmt.Println(s.GetHistory())
			fmt.Printf("----------\n")
		}
	}
}
