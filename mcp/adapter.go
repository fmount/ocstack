package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

// ToolAdapter adapts MCP tools to work with the existing tools system
type ToolAdapter struct {
	client Client
}

// NewToolAdapter creates a new tool adapter
func NewToolAdapter(client Client) *ToolAdapter {
	return &ToolAdapter{client: client}
}

// Tool and related types (to avoid circular dependency)
type Tool struct {
	Type     string    `json:"type"`
	Function *Function `json:"function"`
}

type Function struct {
	Name        string      `json:"name,omitempty"`
	Description string      `json:"description,omitempty"`
	Parameters  *Parameters `json:"parameters,omitempty"`
}

type Parameters struct {
	Type       string                 `json:"type,omitempty"`
	Required   []string               `json:"required,omitempty"`
	Properties map[string]*Properties `json:"properties,omitempty"`
}

type Properties struct {
	Type        string   `json:"type,omitempty"`
	Description string   `json:"description,omitempty"`
	Enum        []string `json:"enum,omitempty"`
}

type FunctionCall struct {
	Name      string         `json:"name"`
	Arguments map[string]any `json:"arguments"`
	Result    string         `json:"result"`
}

// ExecuteMCPTool executes an MCP tool and returns the result in the expected format
func (a *ToolAdapter) ExecuteMCPTool(f interface{}) string {
	functionCall, ok := f.(*FunctionCall)
	if !ok {
		return "Error: invalid function call type"
	}
	if !a.client.IsConnected() {
		return "Error: MCP client not connected"
	}

	ctx := context.Background()
	response, err := a.client.CallTool(ctx, functionCall.Name, functionCall.Arguments)
	if err != nil {
		return fmt.Sprintf("Error calling MCP tool %s: %v", functionCall.Name, err)
	}

	if response.IsError {
		return fmt.Sprintf("MCP tool %s returned error: %s", functionCall.Name, a.formatToolResults(response.Content))
	}

	return a.formatToolResults(response.Content)
}

// formatToolResults converts MCP tool results to string format
func (a *ToolAdapter) formatToolResults(content []ToolResult) string {
	var results []string
	for _, result := range content {
		switch result.Type {
		case "text":
			results = append(results, result.Text)
		default:
			// Handle other types as needed
			results = append(results, result.Text)
		}
	}
	return strings.Join(results, "\n")
}

// MCPToolRegistry manages MCP tools alongside local tools
type MCPToolRegistry struct {
	mcpClient   Client
	localTools  []byte
	mcpEnabled  bool
	adapter     *ToolAdapter
}

// NewMCPToolRegistry creates a new tool registry that can handle both local and MCP tools
func NewMCPToolRegistry() *MCPToolRegistry {
	return &MCPToolRegistry{
		mcpEnabled: false,
	}
}

// SetMCPClient sets the MCP client for the registry
func (r *MCPToolRegistry) SetMCPClient(client Client) {
	r.mcpClient = client
	r.mcpEnabled = client != nil
	if client != nil {
		r.adapter = NewToolAdapter(client)
	}
}

// ExecuteMCPTool executes an MCP tool (implements MCPRegistryInterface)
func (r *MCPToolRegistry) ExecuteMCPTool(f interface{}) string {
	if r.adapter == nil {
		return "Error: MCP adapter not initialized"
	}
	return r.adapter.ExecuteMCPTool(f)
}

// SetLocalTools sets the local tools
func (r *MCPToolRegistry) SetLocalTools(localTools []byte) {
	r.localTools = localTools
}

// GetAllTools returns all available tools (local + MCP) in the format expected by the system
func (r *MCPToolRegistry) GetAllTools() []byte {
	var allTools []Tool

	// Add local tools
	if r.localTools != nil {
		var localToolsList []Tool
		if err := json.Unmarshal(r.localTools, &localToolsList); err == nil {
			allTools = append(allTools, localToolsList...)
		}
	}

	// Add MCP tools if available
	if r.mcpEnabled && r.mcpClient != nil && r.mcpClient.IsConnected() {
		mcpToolsData := r.mcpClient.GetAvailableTools()
		var mcpToolsList []Tool
		if err := json.Unmarshal(mcpToolsData, &mcpToolsList); err == nil {
			allTools = append(allTools, mcpToolsList...)
		}
	}

	result, err := json.Marshal(allTools)
	if err != nil {
		fmt.Printf("Error marshaling combined tools: %v\n", err)
		return r.localTools // fallback to local tools only
	}

	return result
}

// IsToolFromMCP checks if a tool name comes from MCP
func (r *MCPToolRegistry) IsToolFromMCP(toolName string) bool {
	if !r.mcpEnabled || r.mcpClient == nil || !r.mcpClient.IsConnected() {
		return false
	}

	// Check if tool exists in MCP tools
	mcpToolsData := r.mcpClient.GetAvailableTools()
	var mcpTools []Tool
	if err := json.Unmarshal(mcpToolsData, &mcpTools); err != nil {
		return false
	}

	for _, tool := range mcpTools {
		if tool.Function != nil && tool.Function.Name == toolName {
			return true
		}
	}

	return false
}

// Sample MCP configurations for common servers
var (
	// Example configuration for filesystem MCP server (stdio)
	FilesystemMCPConfig = MCPConfig{
		Transport: TransportStdio,
		Command:   []string{"npx", "-y", "@modelcontextprotocol/server-filesystem", "/tmp"},
	}

	// Example configuration for brave search MCP server (stdio)
	BraveSearchMCPConfig = MCPConfig{
		Transport: TransportStdio,
		Command:   []string{"npx", "-y", "@modelcontextprotocol/server-brave-search"},
		Env: map[string]string{
			"BRAVE_API_KEY": "", // Set this to your actual API key
		},
	}

	// Example configuration for SQLite MCP server (stdio)
	SQLiteMCPConfig = MCPConfig{
		Transport: TransportStdio,
		Command:   []string{"npx", "-y", "@modelcontextprotocol/server-sqlite", "--db-path", "/tmp/example.db"},
	}

	// Example configuration for HTTP MCP server
	HTTPMCPConfig = MCPConfig{
		Transport: TransportHTTP,
		ServerURL: "http://localhost:8080/mcp",
	}

	// Example configuration for WebSocket MCP server
	WebSocketMCPConfig = MCPConfig{
		Transport: TransportWebSocket,
		ServerURL: "ws://localhost:8080/mcp",
	}

	// Example configuration for remote HTTP MCP server
	RemoteHTTPMCPConfig = MCPConfig{
		Transport: TransportHTTP,
		ServerURL: "https://api.example.com/mcp",
		Headers: map[string]string{
			"Authorization": "Bearer your-token-here",
		},
	}
)