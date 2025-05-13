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
	"github.com/fmount/ocstack/session"
	"github.com/fmount/ocstack/templates"
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

	// TODO: Select profile
	_, err = templates.LoadProfile("default")
	if err != nil {
		ocstack.ShowWarn(fmt.Sprintf("%s\n", err))
	}

	// Create a new session for the current execution before entering the
	// loop
	s, _ := session.NewSession(
		llm.DefaultModel,
		client,
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
			ocstack.CliCommand(strings.TrimPrefix(q, "/"))
			continue
		}

		// propagate the request to the LLM
		req := &api.GenerateRequest{
			Model: s.Model,
			Prompt: input,
			// set streaming to false
			Stream: new(bool),
		}
		// save the question in the history
		h.Text = append(h.Text, api.Message{
			Role:    "user",
			Content: input,
		})
		// Read the reply
		err = s.GetClient().Generate(ctx, req, &h)
		if err != nil {
			log.Fatal(err)
		}

		//log.Println("[DEBUG] - HISTORY")
		//log.Println(h.Text)

		fmt.Println("---")
	}
}
