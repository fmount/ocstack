package llm

import (
	"github.com/ollama/ollama/api"
)

// Provider - should be used to abstract the LLM provider details (e.g. ollama
// vs something else)
type Provider struct{}

type Session struct {
	Provider  *Provider
	Profile string
	Model   string
	History History
	Tools   []api.Tool
}

type Tool struct {}

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

func (s *Session) NewMessage(input string) []api.Message{
	msg := s.GetHistory().Text
	msg = append(msg, api.Message{
		Role: "user",
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
		Role: "system",
		Content: s.Profile,
	})
}

func NewSession(model string, p *Provider, tmpl string, h History, t []api.Tool) (*Session, error) {
	// we might need some validation and err returning here. Right now this
	// is just a wrapper
	return &Session{
		Provider: p,
		Profile: tmpl,
		Model: model,
		History: h,
		Tools: t,
	}, nil
}

// SaveSession -
func (s *Session) SaveSession() (error) {
	return nil
}

// LoadSession -
func (s *Session) LoadSession() (error, ss *Session) {
	return &Session{}, nil
}
