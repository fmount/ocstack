package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/ollama/ollama/api"
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
			f, err := ToFunctionCall(tool.Function.Name, toolArgs)
			if err != nil {
				return fmt.Errorf(fmt.Sprintf("%v", err))
			}

			//fmt.Printf("[DEBUG] |-> FunctionCall:\n")
			//fmt.Printf("[DEBUG] |-->> %s\n", f.Name)
			//fmt.Printf("[DEBUG] |-->> %v\n", f.Arguments)

			// We do not need this if we're not in debug mode, but let's keep
			// printing the output of the function call
			/*if f.Name == "hello" {
				fmt.Printf("[DEBUG] | -->> %v\n", hello(f.Arguments))
			}*/
			var result string
			if f.Name == "oc" {
				result = oc(f)
				f.Result = result
				fmt.Printf("[DEBUG] | -->> %v\n", result)
			}

			msg := api.Message{
				Role:    "user",
				Content: fmt.Sprintf("Function output is %v\n", result),
			}
			s.UpdateHistory(msg)
			// Process the data we just got by doing a recursive call to the
			// GenerateChat function.
			// TODO: For the future, to avoid an infinite loop, we might want
			// to limit this execution and keep track of the number of times
			// this gets executed.
			outPrompt := fmt.Sprintf("Parse the output received from the function %s called in the previous message with arguments %v\n", f.Name, f.Arguments)
			outPrompt, err = RenderExec(f)
			if err != nil {
				return fmt.Errorf(fmt.Sprintf("%v", err))
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
