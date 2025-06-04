package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
)

const (
	LLAMAPATH = "v1/chat/completions"
)

// LLamaCppProvider -
type LLamaCppProvider struct {
	llamaURL *url.URL
	client   http.Client
}

type LLamaMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type LLamaHistory struct {
	Messages []LLamaMessage `json:"messages"`
}

type LLamaPayload struct {
	Model    string         `json:"model"`
	Messages []LLamaMessage `json:"messages"`
	Stream   bool           `json:"stream"`
	//History History
	Tools []byte `json:"tools"`
}

func (l *LLamaCppProvider) toString() string {
	return fmt.Sprintf("%s://%s/%s", l.llamaURL.Scheme, l.llamaURL.Host, l.llamaURL.Path)
}

// GetLLMClient - implements the interface defined in provider.go
func (p *LLamaCppProvider) GetLLMClient(ctx context.Context) (Client, error) {
	c, err := p.GetLLamaCppClient(ctx)
	if err != nil {
		return c, err
	}
	return c, nil
}

// GetLLamaCppClient - Returns a LLamaCpp client wrapper
func (p *LLamaCppProvider) GetLLamaCppClient(ctx context.Context) (*LLamaCppProvider, error) {
	var llamaURL *url.URL
	var err error

	uRL := os.Getenv("LLAMA_HOST")
	if uRL == "" {
		return nil, fmt.Errorf("Can't find LLAMA_HOST environment variable")
	}
	if llamaURL, err = url.Parse(uRL); err != nil {
		return nil, fmt.Errorf("Malformed LLAMA baseURL")
	}

	prv := &LLamaCppProvider{
		llamaURL: llamaURL,
		client:   http.Client{},
	}
	prv.llamaURL.Path = LLAMAPATH
	return prv, nil
}

// GenerateChat -
func (c *LLamaCppProvider) GenerateChat(
	ctx context.Context,
	input string,
	s *Session,
) error {
	if s.Debug {
		fmt.Printf("Scheme: %s\n", c.llamaURL.Scheme)
		fmt.Printf("Host: %s\n", c.llamaURL.Host)
		fmt.Printf("Path: %s\n", c.llamaURL.Path)
	}

	msg := LLamaMessage{
		Role:    "user",
		Content: input,
	}
	// TODO: append it to the history
	l := LLamaPayload{
		Model:    s.Model,
		Messages: []LLamaMessage{msg},
		Stream:   false,
		Tools:    nil,
	}

	var err error
	var resp []byte
	if resp, err = c.Request(ctx, l, s); err != nil {
		return err
	}
	fmt.Printf("A :> ")
	fmt.Printf("%s\n", resp)
	return nil
}

func (c *LLamaCppProvider) Request(
	ctx context.Context,
	payload LLamaPayload,
	s *Session,
) ([]byte, error) {

	// Marshal the history ([]LLamaMessage)
	// in a json payload
	bMsg, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest(http.MethodPost, c.toString(), bytes.NewBuffer(bMsg))
	if err != nil {
		return nil, fmt.Errorf("could not create request: %s\n", err)
	}
	req.Header.Add("Accept", `application/json`)
	res, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("httpd Request failed: %v", err)
	}
	defer res.Body.Close()

	resBody, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("could not read response: %s\n", err)
	}
	if res.StatusCode != 200 {
		return nil, fmt.Errorf("unexpected status %d\n", res.StatusCode)
	}
	// TODO:
	// - Update history
	// - Function Call if a tool is detected
	// - Render response
	// - return
	return resBody, nil
}
