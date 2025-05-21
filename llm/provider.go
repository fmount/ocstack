package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"text/template"

	"github.com/fmount/ocstack/pkg/ocstack"
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

func NewSession(model string, tmpl string, h History, t []byte) (*Session, error) {
	// we might need some validation and err returning here. Right now this
	// is just a wrapper
	return &Session{
		Profile: tmpl,
		Model:   model,
		History: h,
		Tools:   t,
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

type Tool struct {
	Type     string    `json:"type"`
	Function *Function `json:"function"`
}

type FunctionCall struct {
	Name      string         `json:"name"`
	Arguments map[string]any `json:"arguments"`
	Result    string         `json:"result"`
}

type Properties struct {
	Type        string   `json:"type,omitempty"`
	Description string   `json:"description,omitempty"`
	Enum        []string `json:"enum,omitempty"`
}

type Parameters struct {
	Type       string                 `json:"type,omitempty"`
	Required   []string               `json:"required,omitempty"`
	Properties map[string]*Properties `json:"properties,omitempty"`
}

type Function struct {
	Name        string      `json:"name,omitempty"`
	Description string      `json:"description,omitempty"`
	Parameters  *Parameters `json:"parameters,omitempty"`
}

// ToFunctionArgs -
func ToFunctionArgs(b []byte) (map[string]any, error) {
	m := make(map[string]any)
	err := json.Unmarshal(b, &m)
	if err != nil {
		return nil, fmt.Errorf("Can't unmarshal data")
	}
	return m, nil
}

func ToFunctionCall(name string, b []byte) (*FunctionCall, error) {
	var err error
	var args map[string]any
	if args, err = ToFunctionArgs(b); err != nil {
		return nil, err
	}
	f := &FunctionCall{
		Name:      name,
		Arguments: args,
	}
	return f, nil
}

func RenderExec(f *FunctionCall) (string, error) {
	tmpl, err := template.ParseFiles("template/resources/execResult.tmpl")
	if err != nil {
		return "", fmt.Errorf("Error parsing template file: %v", err)
	}
	var buf bytes.Buffer
	// Execute the template with the data and write the output to stdout
	err = tmpl.Execute(&buf, f)
	if err != nil {
		return "", fmt.Errorf("Error executing template: %v", err)
	}
	return buf.String(), nil
}

/**
 * Function definition
**/
func hello(args map[string]any) string {
	return fmt.Sprintf("Hello %s\n", args["name"])
}

func oc(f *FunctionCall) string {
	res, _ := ocstack.ExecTool(f.Name, "get nodes --show-labels")
	return res.ToString()
}
