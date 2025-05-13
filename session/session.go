package session

import (
	"github.com/ollama/ollama/api"
	"github.com/fmount/ocstack/llm"
)

type Session struct {
	Client  *llm.OllamaClient
	Model   string
	History llm.History
	Tools   []api.Tool
}

func NewSession(
	model string,
	c *llm.OllamaClient,
	h llm.History,
	t []api.Tool,
) (*Session, error) {
	// we might need some validation and err returning here. Right now this
	// is just a wrapper
	return &Session{
		Client: c,
		Model: model,
		History: h,
		Tools: t,
	}, nil
}

func (s *Session) GetClient() *llm.OllamaClient {
	return s.Client
}

// SaveSession -
func (s *Session) SaveSession() (error) {
	return nil
}

// LoadSession -
func (s *Session) LoadSession() (error, ss *Session) {
	return &Session{}, nil
}
