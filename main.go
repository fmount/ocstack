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
	"github.com/ollama/ollama/api"
)

func main() {
	// Move this part into a tty.header()
	fmt.Println("----")
	fmt.Println("Hello, ocstack!")
	fmt.Println("----")
	fmt.Println("I :> Run /help to get a list of available commands")

	ctx := context.Background()

	client, err := llm.GetOllamaClient(ctx)
	if err != nil {
		log.Fatal(err)
	}

	h := session.History{}

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
			Model: llm.DefaultModel,
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
		err = client.Generate(ctx, req, &h)
		if err != nil {
			log.Fatal(err)
		}

		//log.Println("[DEBUG] - HISTORY")
		//log.Println(h.Text)

		fmt.Println("---")
	}
}
