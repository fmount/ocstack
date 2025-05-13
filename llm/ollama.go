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

type History struct {
	Text []api.Message
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

func (c *OllamaClient) Models(ctx context.Context) ([]string, error) {
	// TODO: implement this at some point
	return []string{}, nil
}
