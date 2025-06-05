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

// LLamaMessage represents the message content
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

// ChatCompletion represents the main response structure
type LLamaChatCompletion struct {
	ID                string        `json:"id"`
	Object            string        `json:"object"`
	Created           int64         `json:"created"`
	Model             string        `json:"model"`
	SystemFingerprint string        `json:"system_fingerprint"`
	Choices           []LLamaChoice `json:"choices"`
	Usage             LLamaUsage    `json:"usage"`
	Timings           LLamaTimings  `json:"timings"`
}

// LLamaChoice represents each choice in the response
type LLamaChoice struct {
	Index        int          `json:"index"`
	Message      LLamaMessage `json:"message"`
	FinishReason string       `json:"finish_reason"`
}

// LLamaUsage represents token usage statistics
type LLamaUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// LLamaTimings represents performance metrics
type LLamaTimings struct {
	PromptN             int     `json:"prompt_n"`
	PromptMs            float64 `json:"prompt_ms"`
	PromptPerTokenMs    float64 `json:"prompt_per_token_ms"`
	PromptPerSecond     float64 `json:"prompt_per_second"`
	PredictedN          int     `json:"predicted_n"`
	PredictedMs         float64 `json:"predicted_ms"`
	PredictedPerTokenMs float64 `json:"predicted_per_token_ms"`
	PredictedPerSecond  float64 `json:"predicted_per_second"`
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
		fmt.Printf("[DEBUG] - Scheme: %s\n", c.llamaURL.Scheme)
		fmt.Printf("[DEBUG] - Host: %s\n", c.llamaURL.Host)
		fmt.Printf("[DEBUG] - Path: %s\n", c.llamaURL.Path)
	}

	h := s.GetHistory()
	var msgs []LLamaMessage

	// Convert []interface{} to []LLamaMessage
	for _, item := range h.Text {
		if msg, ok := item.Text.(LLamaMessage); ok {
			msgs = append(msgs, msg)
		}
	}

	msgs = append(msgs, LLamaMessage{
		Role:    "user",
		Content: input,
	})

	l := LLamaPayload{
		Model:    s.Model,
		Messages: msgs,
		Stream:   false,
		Tools:    nil,
	}

	var err error
	var resp []byte
	if resp, err = c.Request(ctx, l, s); err != nil {
		return err
	}

	// Parse the resulting JSON
	var completion LLamaChatCompletion
	err = json.Unmarshal([]byte(resp), &completion)
	if err != nil {
		return err
	}
	fmt.Printf("A :> ")
	// Print response only if there's at least one available
	// choice
	if len(completion.Choices) > 0 {
		var result string
		result = completion.Choices[0].Message.Content
		fmt.Printf("%s\n", result)
		s.UpdateHistory(Message{
			Role: "assistant",
			Text: result,
		})
	}

	// TODO:
	// - Function Call if a tool is detected
	// - Render response using the golang template

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
	return resBody, nil
}
