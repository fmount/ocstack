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
	"time"

	"github.com/fmount/ocstack/pkg/ocstack"
	"github.com/fmount/ocstack/tools"
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
	Role      string          `json:"role"`
	Content   string          `json:"content"`
	ToolCalls []LLamaToolCall `json:"tool_calls,omitempty"`
}

type LLamaHistory struct {
	Messages []LLamaMessage `json:"messages"`
}

type LLamaPayload struct {
	Model    string         `json:"model"`
	Messages []LLamaMessage `json:"messages"`
	Stream   bool           `json:"stream"`
	//History History
	Tools []tools.Tool `json:"tools"`
}

// ToLLamaCppTools -
func ToLLamaCppTools(b []byte) ([]tools.Tool, error) {
	var t []tools.Tool
	err := json.Unmarshal(b, &t)
	if err != nil {
		return nil, err
	}
	return t, err
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

// LLamaToolCall represents a tool call in the response
type LLamaToolCall struct {
	ID       string              `json:"id"`
	Type     string              `json:"type"`
	Function LLamaFunctionCall   `json:"function"`
}

// LLamaFunctionCall represents the function call details
type LLamaFunctionCall struct {
	Name      string                 `json:"name"`
	Arguments map[string]interface{} `json:"arguments"`
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
		client: http.Client{
			Timeout: 120 * time.Second,
		},
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

	// If it's the first message, let's set some context in the history
	// to drive the reasoning
	if len(s.GetHistory().Text) == 0 {
		s.UpdateContext()
	}

	var er error
	var t []tools.Tool = []tools.Tool{}

	t, er = ToLLamaCppTools(s.Tools)
	if er != nil {
		return er
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
		Tools:    t,
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

	var lastLLMResponse string

	fmt.Printf("A :> ")
	// Print response only if there's at least one available choice
	if len(completion.Choices) > 0 {
		var result string
		result = completion.Choices[0].Message.Content
		fmt.Printf("%s\n", result)

		// Store LLM response for action detection
		lastLLMResponse = result

		s.UpdateHistory(Message{
			Role: "assistant",
			Text: result,
		})

		// Check for recommendations in LLM response (both direct and collective)
		CheckForRecommendations(s, lastLLMResponse)

		// Process tool calls if present
		if len(completion.Choices[0].Message.ToolCalls) > 0 {
			fmt.Printf("T :> ")
			fmt.Println(completion.Choices[0].Message.ToolCalls)

			// Collect all tool results before processing them collectively
			var toolResults []*tools.FunctionCall
			ns := s.GetConfig()[ocstack.NAMESPACE]

			for _, toolCall := range completion.Choices[0].Message.ToolCalls {
				// Build function Call
				toolArgs, err := json.Marshal(toolCall.Function.Arguments)
				if err != nil {
					return fmt.Errorf("Error marshaling args")
				}
				f, err := tools.ToFunctionCall(toolCall.Function.Name, toolArgs)
				if err != nil {
					return fmt.Errorf("%v", err)
				}

				var toolResult string

				// MCP tools take priority - check MCP first
				if mcpRegistry := s.GetMCPRegistry(); mcpRegistry != nil && mcpRegistry.IsToolFromMCP(f.Name) {
					// ALWAYS override namespace parameter with ocstack's configured namespace
					// This ensures ocstack's namespace setting takes precedence over LLM-provided values
					if f.Arguments == nil {
						f.Arguments = make(map[string]interface{})
					}
					f.Arguments["namespace"] = ns
					// Execute MCP tool (preferred)
					toolResult = mcpRegistry.ExecuteMCPTool(f)
					f.Result = toolResult
				} else {
					// Tool not found in MCP
					toolResult = fmt.Sprintf("Tool '%s' not found in MCP", f.Name)
					f.Result = toolResult
				}

				if s.Debug {
					fmt.Printf("[DEBUG] |-->> %s\n", f.Name)
					fmt.Printf("[DEBUG] | -->> out: %s\n", f.Result)
				}

				// Add to collection instead of processing immediately
				toolResults = append(toolResults, f)
			}

			// Process all tool results collectively for agentic reasoning
			if len(toolResults) > 0 {
				collectivePrompt := tools.RenderCollectiveExec(toolResults)
				s.ProcessingCollective = true
				c.GenerateChat(ctx, collectivePrompt, s)
				s.ProcessingCollective = false
			}
		}
	}

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
	req.Header.Add("Content-Type", `application/json`)
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
	if s.Debug {
		fmt.Printf("[DEBUG] - JSON Response -> %s\n", resBody)
	}
	
	// Add timeout context to prevent hanging
	if res.StatusCode == 200 && len(resBody) == 0 {
		return nil, fmt.Errorf("empty response from server")
	}
	return resBody, nil
}
