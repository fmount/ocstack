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
	"github.com/fmount/ocstack/template"
	"github.com/ollama/ollama/api"
)

func main() {

	// Validate ocstack input required to access Tools
	ocstack.ExitOnErrors()

	ctx := context.Background()
	client, err := llm.GetOllamaClient(ctx)
	if err != nil {
		log.Fatal(err)
	}

	h := llm.History{}
	t := []api.Tool{}
	p := &llm.Provider{}

	profile, err := templates.LoadProfile("default")
	if err != nil {
		ocstack.ShowWarn(fmt.Sprintf("%s\n", err))
	}

	// Create a new session for the current execution before entering the
	// loop
	s, _ := llm.NewSession(
		llm.DefaultModel,
		p,
		profile,
		h,
		t,
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

		// process potential commands
		if len(input) > 0 && strings.HasPrefix(input, "/") {
			// Trim any whitespace from the input
			q := strings.TrimSpace(input)
			ocstack.CliCommand(strings.TrimPrefix(q, "/"), s)
			continue
		}

		// propagate the request to the LLM
		err = client.GenerateChat(
			ctx,
			input,
			s,
		)

		/*
		fmt.Println("------------------")
		log.Println("[DEBUG] - HISTORY")
		log.Println(s.GetHistory().Text)
		fmt.Println("------------------")
		fmt.Println("---")
		*/
	}
}
