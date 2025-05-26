package llm

import (
	"context"
	"github.com/ollama/ollama/api"
)

const (
	OLLAMAPROVIDER = "ollama"
)

type Client interface {
	GenerateChat(c context.Context, input string, s *Session) error
}

// Provider - should be used to abstract the LLM provider details (e.g. ollama
// vs something else)
type Provider interface {
	GetLLMClient(c context.Context) (Client, error)
}

// GetProvider - based on what is passed, it returns a new LLM Client
func GetProvider(pID string) (Client, error) {
	switch pID {
	case OLLAMAPROVIDER:
		var p OllamaProvider
		client, err := p.GetLLMClient(context.Background())
		if err != nil {
			return nil, err
		}
		return client, err
	default:
		return nil, nil
	}
}

type Session struct {
	Profile string
	Model   string
	History History
	Tools   []byte
	Debug   bool
}

type History struct {
	Text []api.Message
}

func (s *Session) GetHistory() History {
	return s.History
}

func (s *Session) SetHistory(h History) {
	s.History = h
}

func (s *Session) GetProfile() string {
	return s.Profile
}

func (s *Session) NewMessage(input string) []api.Message {
	msg := s.GetHistory().Text
	msg = append(msg, api.Message{
		Role:    "user",
		Content: input,
	})
	return msg
}

func (s *Session) UpdateHistory(m api.Message) {
	h := s.GetHistory().Text
	h = append(h, m)
	s.SetHistory(History{h})
}

func (s *Session) SetContext() {
	s.UpdateHistory(api.Message{
		Role:    "system",
		Content: s.Profile,
	})
}

func NewSession(model string, tmpl string, h History, t []byte, d bool) (*Session, error) {
	// we might need some validation and err returning here. Right now this
	// is just a wrapper
	return &Session{
		Profile: tmpl,
		Model:   model,
		History: h,
		Tools:   t,
		Debug: d,
	}, nil
}

// SaveSession -
func (s *Session) SaveSession() error {
	return nil
}

// LoadSession -
func (s *Session) LoadSession() (error, ss *Session) {
	return &Session{}, nil
}
