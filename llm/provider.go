package llm

import (
	"context"
	"fmt"
	"strings"
)

// MCPRegistryInterface defines the interface for MCP tool registry
type MCPRegistryInterface interface {
	IsToolFromMCP(string) bool
	ExecuteMCPTool(interface{}) string
	GetAllTools() []byte
}

const (
	OLLAMAPROVIDER = "ollama"
	LLAMACPP       = "llama"
)

type Client interface {
	GenerateChat(c context.Context, input string, s *Session) error
}

// CheckForRecommendations is a helper method that should be called after LLM response
func CheckForRecommendations(s *Session, response string) {
	if action := s.DetectRecommendedAction(response); action != nil && s.PendingAction == nil {
		s.State = StateAwaitingConfirmation
		s.PendingAction = action
		fmt.Printf("\nRecommended Action: %s\n", action.Description)
		fmt.Printf("Would you like to proceed? (y/n): ")
	}
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

// SessionState represents the current state of the agentic workflow
type SessionState string

const (
	StateNormal               SessionState = "normal"
	StateAwaitingConfirmation SessionState = "awaiting_confirmation"
	StateExecuting            SessionState = "executing"
)

// PendingAction represents an action waiting for user confirmation
type PendingAction struct {
	Type        string                 `json:"type"`
	Description string                 `json:"description"`
	Parameters  map[string]interface{} `json:"parameters"`
}

type Session struct {
	Profile              string
	Model                string
	History              History
	Tools                []byte
	Debug                bool
	Config               map[string]string
	mcpRegistry          interface{} // Interface to avoid circular dependency
	State                SessionState
	PendingAction        *PendingAction
	ProcessingCollective bool
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
		Profile:              tmpl,
		Model:                model,
		History:              h,
		Tools:                t,
		Debug:                d,
		Config:               c,
		mcpRegistry:          nil,
		State:                StateNormal,
		PendingAction:        nil,
		ProcessingCollective: false,
	}, nil
}

// SetMCPRegistry sets the MCP registry for the session
func (s *Session) SetMCPRegistry(registry interface{}) {
	s.mcpRegistry = registry
}

// GetMCPRegistry returns the MCP registry if it implements the required interface
func (s *Session) GetMCPRegistry() MCPRegistryInterface {
	if registry, ok := s.mcpRegistry.(MCPRegistryInterface); ok {
		return registry
	}
	return nil
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

// DetectRecommendedAction analyzes LLM response using template-based structured parsing
func (s *Session) DetectRecommendedAction(response string) *PendingAction {
	if response == "" {
		return nil
	}

	// Parse the structured ## Recommendations section from the template
	recommendation := parseRecommendationSection(response)
	if recommendation == "" {
		return nil
	}

	return &PendingAction{
		Type:        "execute_recommendation",
		Description: recommendation,
		Parameters: map[string]interface{}{
			"recommendation": recommendation,
		},
	}
}

// parseRecommendationSection extracts actionable recommendations from the structured template response
func parseRecommendationSection(response string) string {
	lines := strings.Split(response, "\n")

	// Find the ## Recommendations section
	inRecommendationSection := false
	recommendationLines := []string{}

	for _, line := range lines {
		trimmedLine := strings.TrimSpace(line)

		// Check for the start of the Recommendations section (## or ###)
		if strings.HasPrefix(trimmedLine, "### Recommendations") || strings.HasPrefix(trimmedLine, "## Recommendations") {
			inRecommendationSection = true
			continue
		}

		// Check for the end of the section (next ## or ### header)
		if inRecommendationSection && (strings.HasPrefix(trimmedLine, "###") || strings.HasPrefix(trimmedLine, "##")) {
			break
		}

		// Collect recommendation content
		if inRecommendationSection && trimmedLine != "" {
			recommendationLines = append(recommendationLines, trimmedLine)
		}
	}

	// Process the collected recommendation content
	if len(recommendationLines) == 0 {
		return ""
	}

	recommendationText := strings.Join(recommendationLines, " ")
	recommendationText = strings.TrimSpace(recommendationText)

	// Check if the recommendation is "None" (case-insensitive)
	if strings.ToLower(recommendationText) == "none" {
		return ""
	}

	// Clean up common formatting artifacts
	recommendationText = strings.TrimPrefix(recommendationText, "[")
	recommendationText = strings.TrimSuffix(recommendationText, "]")
	recommendationText = strings.TrimSpace(recommendationText)

	if recommendationText == "" {
		return ""
	}

	return recommendationText
}

// HandleConfirmation handles user confirmation for pending actions
func (s *Session) HandleConfirmation(input string, client Client, ctx context.Context) {
	if s.PendingAction == nil {
		fmt.Println("No pending action to confirm")
		s.State = StateNormal
		return
	}

	input = strings.ToLower(strings.TrimSpace(input))
	if input == "y" || input == "yes" {
		fmt.Printf("Executing: %s\n", s.PendingAction.Description)

		// Execute the pending action
		switch s.PendingAction.Type {
		case "execute_recommendation":
			s.executeRecommendation(client, ctx)
		default:
			fmt.Printf("Unknown action type: %s\n", s.PendingAction.Type)
		}

	} else if input == "n" || input == "no" {
		fmt.Println("Action cancelled")
	} else {
		fmt.Println("Please respond with 'y' or 'n'")
		return // Don't reset state, wait for valid input
	}

	// Reset session state
	s.State = StateNormal
	s.PendingAction = nil
}

// executeRecommendation feeds the recommendation back to LLM for execution
func (s *Session) executeRecommendation(client Client, ctx context.Context) {
	if s.PendingAction == nil {
		fmt.Println("No pending action to execute")
		return
	}

	// Extract the original recommendation
	recommendation, exists := s.PendingAction.Parameters["recommendation"].(string)
	if !exists {
		fmt.Println("No recommendation found in pending action")
		return
	}

	// Feed the recommendation back to the LLM for processing
	err := client.GenerateChat(ctx, recommendation, s)
	if err != nil {
		fmt.Printf("Error executing recommendation: %v\n", err)
	}
}
