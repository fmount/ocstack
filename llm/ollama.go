package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/ollama/ollama/api"
	tools "github.com/fmount/ocstack/tools"
	//"os"
)

const (
	DefaultModel = "gemma2:latest"
	LLAMA        = "llama3"
	QWEN         = "qwen2.5:1.5b"
)

type OllamaProvider struct {
	client api.Client
}

// GetLLMClient - implements the interface defined in provider.go
func (p *OllamaProvider) GetLLMClient(ctx context.Context) (Client, error) {
	c, err := p.GetOllamaClient(ctx)
	if err != nil {
		return c, err
	}
	return c, nil
}

// GetOllamaClient - Returns the api.Client wrapper
func (p *OllamaProvider) GetOllamaClient(ctx context.Context) (*OllamaProvider, error) {
	c, err := api.ClientFromEnvironment()
	if err != nil {
		return nil, err
	}

	return &OllamaProvider{
		client: *c,
	}, nil
}

// Models -
func (c *OllamaProvider) Models(ctx context.Context) ([]string, error) {
	// TODO: implement this at some point
	return []string{}, nil
}

// Generate -
func (c *OllamaProvider) Generate(
	ctx context.Context,
	req *api.GenerateRequest,
	history *History,
) error {

	respFunc := func(resp api.GenerateResponse) error {
		// Only print the response here; GenerateResponse has a number of other
		// interesting fields you want to examine.
		fmt.Println(resp.Response)
		if history != nil {
			history.Text = append(history.Text, api.Message{
				Role:    "user",
				Content: resp.Response,
			})
		}
		return nil
	}
	c.client.Generate(ctx, req, respFunc)
	return nil
}

// GenerateChat -
func (c *OllamaProvider) GenerateChat(
	ctx context.Context,
	input string,
	s *Session,
) error {
	// If it's the first message, let's set some context in the history
	// to drive the reasoning
	if len(s.GetHistory().Text) == 0 {
		s.UpdateHistory(api.Message{
			Role:    "system",
			Content: s.Profile,
		})
	}

	// Process user input and build a message that can be passed to
	// a ChatRequest
	msg := s.GetHistory().Text
	msg = append(msg, api.Message{
		Role:    "user",
		Content: input,
	})

	// Build ollama tools struct
	t, erro := s.ToOllamaTools(s.Tools)
	if erro != nil {
		return fmt.Errorf("Can't get tools")
	}

	req := &api.ChatRequest{
		Model:    s.Model,
		Messages: msg,
		Stream:   new(bool),
		Tools:    t,
	}

	respFunc := func(resp api.ChatResponse) error {
		fmt.Printf("A :> ")
		fmt.Println(resp.Message.Content)
		fmt.Printf("T :> ")
		fmt.Println(resp.Message.ToolCalls)
		// Check if content is empty (e.g. it returned a ToolCall)
		msg := api.Message{
			Role:    "user",
			Content: resp.Message.Content,
		}
		s.UpdateHistory(msg)

		for _, tool := range resp.Message.ToolCalls {
			// Build function Call
			toolArgs, err := json.Marshal(tool.Function.Arguments)
			if err != nil {
				return fmt.Errorf("Error marshaling args")
			}
			f, err := tools.ToFunctionCall(tool.Function.Name, toolArgs)
			if err != nil {
				return fmt.Errorf("%v", err)
			}

			var result string

			if f.Name == "hello" {
				result = tools.Hello(f.Arguments)
				f.Result = result
			}

			if f.Name == "oc" {
				result = tools.OC(f)
				f.Result = result
				//os.Exit(0)
			}

			if s.Debug {
				fmt.Printf("[DEBUG] |-> FunctionCall:\n")
				fmt.Printf("[DEBUG] |-->> %s\n", f.Name)
				fmt.Printf("[DEBUG] |-->> %v\n", f.Arguments)
				fmt.Printf("[DEBUG] | -->> %v\n", f.Result)
			}
			// Process the data we just got by doing a recursive call to the
			// GenerateChat function.
			outPrompt, err := tools.RenderExec(f)
			if err != nil {
				return fmt.Errorf("%v", err)
			}
			c.GenerateChat(ctx, outPrompt, s)
		}
		return nil
	}

	err := c.client.Chat(ctx, req, respFunc)
	if err != nil {
		return err
	}
	return nil
}

// ToOllamaTools -
func (p *Session) ToOllamaTools(b []byte) ([]api.Tool, error) {
	var tools api.Tools
	err := json.Unmarshal(b, &tools)
	if err != nil {
		return nil, err
	}
	return tools, err
}
