package llm

import (
	"context"
	"fmt"
)

const (
	OLLAMAPROVIDER = "ollama"
	LLAMACPP       = "llama"
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
	case LLAMACPP:
		var p LLamaCppProvider
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
	Config  map[string]string
}

type Message struct {
	Role string
	Text interface{}
}

// History is a list of messages associated with a given session
type History struct {
	Text []Message
}

// GetHistory -
func (s *Session) GetHistory() History {
	return s.History
}

// SetHistory -
func (s *Session) SetHistory(h History) {
	s.History = h
}

// GetProfile -
func (s *Session) GetProfile() string {
	return s.Profile
}

// UpdateHistory -
func (s *Session) UpdateHistory(m Message) {
	h := s.GetHistory().Text
	h = append(h, m)
	s.SetHistory(History{h})
}

func (s *Session) GetConfig() map[string]string {
	return s.Config
}

func (s *Session) SetConfig(k string, v string) {
	s.Config[k] = v
}

func (s *Session) GetConfigItem(k string) (string, string) {

	val, ok := s.Config[k]
	if !ok {
		return "", "Config option not present"
	}
	return val, ""
}

func (s *Session) ShowConfig() {
	for k, v := range s.Config {
		fmt.Printf("[%s - %s]\n", k, v)
	}
}

// NewSession -
func NewSession(model string, tmpl string, h History, t []byte, d bool, c map[string]string) (*Session, error) {
	// we might need some validation and err returning here. Right now this
	// is just a wrapper
	return &Session{
		Profile: tmpl,
		Model:   model,
		History: h,
		Tools:   t,
		Debug:   d,
		Config:  c,
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

// UpdateContext -
func (s *Session) UpdateContext() {
	s.UpdateHistory(Message{
		Role: "system",
		Text: s.Profile,
	})
}
