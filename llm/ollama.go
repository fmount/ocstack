package llm

import (
	"context"
	"fmt"
	"github.com/ollama/ollama/api"
)

const (
	DefaultModel = "gemma2:latest"
)

type OllamaClient struct {
	client api.Client
}

func GetOllamaClient(ctx context.Context) (*OllamaClient, error) {
	c, err := api.ClientFromEnvironment()
	if err != nil {
		return nil, err
	}

	return &OllamaClient{
		client: *c,
	}, nil
}

func (c *OllamaClient) Generate(
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

func (c *OllamaClient) GenerateChat(
	ctx context.Context,
	input string,
	s *Session,
) error {
	// If it's the first message, let's set some context in the history
	// to drive the reasoning
	if len(s.GetHistory().Text) == 0 {
		s.UpdateHistory(api.Message{
			Role: "system",
			Content: s.Profile,
		})
	}

	// Process user input and build a message that can be passed to
	// a ChatRequest
	msg := s.GetHistory().Text
	msg = append(msg, api.Message{
		Role: "user",
		Content: input,
	})

	req := &api.ChatRequest{
		Model:    s.Model,
		Messages: msg,
		// set streaming to false
		Stream: new(bool),
		Tools:  s.Tools,
	}

	respFunc := func(resp api.ChatResponse) error {
		fmt.Println(resp.Message.Content)
		fmt.Println(resp.Message.ToolCalls)
		msg := api.Message{
			Role: "user",
			Content: resp.Message.Content,
		}
		s.UpdateHistory(msg)
		return nil
	}

	err := c.client.Chat(ctx, req, respFunc)
	if err != nil {
		return err
	}
	return nil
}

func (c *OllamaClient) Models(ctx context.Context) ([]string, error) {
	// TODO: implement this at some point
	return []string{}, nil
}
