package mcp

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"sync"
	"time"
)

// Client interface defines the MCP client operations
type Client interface {
	Connect(ctx context.Context) error
	Disconnect() error
	ListTools(ctx context.Context) ([]MCPTool, error)
	CallTool(ctx context.Context, name string, args map[string]interface{}) (*CallToolResponse, error)
	GetAvailableTools() []byte // Returns tools in the format expected by your existing system
	IsConnected() bool
}

// MCPClient implements the MCP client
type MCPClient struct {
	config       MCPConfig
	state        ConnectionState
	serverInfo   *ServerInfo
	capabilities *ServerCapabilities
	tools        []MCPTool
	
	// Transport abstraction
	transport Transport
	
	// JSON-RPC for stdio transport
	requestID int
	responses map[interface{}]chan JSONRPCResponse
	mu        sync.RWMutex
	
	// Context and cancellation
	ctx    context.Context
	cancel context.CancelFunc
}

// NewClient creates a new MCP client
func NewClient(config MCPConfig) *MCPClient {
	if config.Timeout == 0 {
		config.Timeout = 30 * time.Second
	}
	if config.MaxRetries == 0 {
		config.MaxRetries = 3
	}
	
	// Set default transport if not specified
	if config.Transport == "" {
		if config.ServerURL != "" {
			config.Transport = TransportHTTP
		} else {
			config.Transport = TransportStdio
		}
	}
	
	return &MCPClient{
		config:    config,
		state:     StateDisconnected,
		responses: make(map[interface{}]chan JSONRPCResponse),
	}
}

// Connect establishes connection to the MCP server
func (c *MCPClient) Connect(ctx context.Context) error {
	if c.state != StateDisconnected {
		return fmt.Errorf("client already connected or connecting")
	}
	
	c.setState(StateConnecting)
	
	// Create context with timeout
	c.ctx, c.cancel = context.WithCancel(ctx)
	
	// Create and connect transport
	if err := c.createTransport(); err != nil {
		c.setState(StateDisconnected)
		return fmt.Errorf("failed to create transport: %w", err)
	}
	
	if err := c.transport.Connect(c.ctx); err != nil {
		c.setState(StateDisconnected)
		return fmt.Errorf("failed to connect transport: %w", err)
	}
	
	// Start message handling for stdio transport only
	if c.config.Transport == TransportStdio {
		go c.handleMessages()
	}
	
	// Initialize MCP protocol
	if err := c.initialize(); err != nil {
		c.setState(StateDisconnected)
		return fmt.Errorf("failed to initialize MCP protocol: %w", err)
	}
	
	// List available tools
	if err := c.refreshTools(); err != nil {
		// Don't fail connection if tool listing fails, just log
		fmt.Printf("Warning: failed to refresh tools: %v\n", err)
	}
	
	c.setState(StateConnected)
	return nil
}

// Disconnect closes the connection to the MCP server
func (c *MCPClient) Disconnect() error {
	if c.state == StateDisconnected || c.state == StateClosed {
		return nil
	}
	
	c.setState(StateClosed)
	
	// Cancel context
	if c.cancel != nil {
		c.cancel()
	}
	
	// Disconnect transport
	if c.transport != nil {
		c.transport.Disconnect()
	}
	
	return nil
}

// IsConnected returns true if the client is connected
func (c *MCPClient) IsConnected() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.state == StateConnected
}

// ListTools returns the available tools from the MCP server
func (c *MCPClient) ListTools(ctx context.Context) ([]MCPTool, error) {
	if !c.IsConnected() {
		return nil, fmt.Errorf("client not connected")
	}
	
	request := JSONRPCRequest{
		JSONRpc: "2.0",
		ID:      c.nextRequestID(),
		Method:  "tools/list",
		Params:  ListToolsRequest{},
	}
	
	response, err := c.sendRequest(ctx, request)
	if err != nil {
		return nil, fmt.Errorf("failed to send tools/list request: %w", err)
	}
	
	if response.Error != nil {
		return nil, fmt.Errorf("tools/list failed: %s", response.Error.Message)
	}
	
	var toolsResponse ListToolsResponse
	resultBytes, err := json.Marshal(response.Result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal tools response: %w", err)
	}
	
	if err := json.Unmarshal(resultBytes, &toolsResponse); err != nil {
		return nil, fmt.Errorf("failed to unmarshal tools response: %w", err)
	}
	
	return toolsResponse.Tools, nil
}

// CallTool executes a tool on the MCP server
func (c *MCPClient) CallTool(ctx context.Context, name string, args map[string]interface{}) (*CallToolResponse, error) {
	if !c.IsConnected() {
		return nil, fmt.Errorf("client not connected")
	}
	
	request := JSONRPCRequest{
		JSONRpc: "2.0",
		ID:      c.nextRequestID(),
		Method:  "tools/call",
		Params: CallToolRequest{
			Name:      name,
			Arguments: args,
		},
	}
	
	response, err := c.sendRequest(ctx, request)
	if err != nil {
		return nil, fmt.Errorf("failed to send tools/call request: %w", err)
	}
	
	if response.Error != nil {
		return nil, fmt.Errorf("tools/call failed: %s", response.Error.Message)
	}
	
	var callResponse CallToolResponse
	resultBytes, err := json.Marshal(response.Result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal call response: %w", err)
	}
	
	if err := json.Unmarshal(resultBytes, &callResponse); err != nil {
		return nil, fmt.Errorf("failed to unmarshal call response: %w", err)
	}
	
	return &callResponse, nil
}

// GetAvailableTools returns tools in the format expected by your existing system
func (c *MCPClient) GetAvailableTools() []byte {
	c.mu.RLock()
	defer c.mu.RUnlock()
	
	var convertedTools []Tool
	
	for _, mcpTool := range c.tools {
		tool := Tool{
			Type: "function",
			Function: &Function{
				Name:        mcpTool.Name,
				Description: mcpTool.Description,
				Parameters:  c.convertSchema(mcpTool.InputSchema),
			},
		}
		convertedTools = append(convertedTools, tool)
	}
	
	data, err := json.Marshal(convertedTools)
	if err != nil {
		fmt.Printf("Error marshaling tools: %v\n", err)
		return []byte("[]")
	}
	
	return data
}

// Private methods

func (c *MCPClient) setState(state ConnectionState) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.state = state
}

func (c *MCPClient) createTransport() error {
	switch c.config.Transport {
	case TransportHTTP:
		if c.config.ServerURL == "" {
			return fmt.Errorf("ServerURL required for HTTP transport")
		}
		c.transport = NewHTTPTransport(c.config.ServerURL, c.config.Timeout)
		
	case TransportWebSocket:
		if c.config.ServerURL == "" {
			return fmt.Errorf("ServerURL required for WebSocket transport")
		}
		// Convert HTTP URL to WebSocket URL if needed
		wsURL := c.config.ServerURL
		if strings.HasPrefix(wsURL, "http://") {
			wsURL = strings.Replace(wsURL, "http://", "ws://", 1)
		} else if strings.HasPrefix(wsURL, "https://") {
			wsURL = strings.Replace(wsURL, "https://", "wss://", 1)
		}
		c.transport = NewWebSocketTransport(wsURL)
		
	case TransportStdio:
		if len(c.config.Command) == 0 {
			return fmt.Errorf("Command required for stdio transport")
		}
		// For stdio, we'll create a legacy transport that wraps the existing logic
		return c.createStdioTransport()
		
	default:
		return fmt.Errorf("unsupported transport type: %s", c.config.Transport)
	}
	
	return nil
}

func (c *MCPClient) createStdioTransport() error {
	// This is a temporary implementation - the stdio logic should be properly 
	// moved to StdioTransport struct
	cmd := exec.CommandContext(c.ctx, c.config.Command[0], c.config.Command[1:]...)
	
	// Set environment variables
	if c.config.Env != nil {
		for key, value := range c.config.Env {
			cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", key, value))
		}
	}
	
	var err error
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdin pipe: %w", err)
	}
	
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdout pipe: %w", err)
	}
	
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("failed to create stderr pipe: %w", err)
	}
	
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start command: %w", err)
	}
	
	// Create a simple stdio transport wrapper
	c.transport = &stdioTransportWrapper{
		cmd:    cmd,
		stdin:  stdin,
		stdout: stdout,
		stderr: stderr,
	}
	
	return nil
}

// Temporary wrapper for stdio transport
type stdioTransportWrapper struct {
	cmd    *exec.Cmd
	stdin  io.WriteCloser
	stdout io.ReadCloser
	stderr io.ReadCloser
}

func (s *stdioTransportWrapper) Connect(ctx context.Context) error {
	// Already connected when created
	return nil
}

func (s *stdioTransportWrapper) Disconnect() error {
	if s.stdin != nil {
		s.stdin.Close()
	}
	if s.stdout != nil {
		s.stdout.Close()
	}
	if s.stderr != nil {
		s.stderr.Close()
	}
	if s.cmd != nil && s.cmd.Process != nil {
		s.cmd.Process.Kill()
		s.cmd.Wait()
	}
	return nil
}

func (s *stdioTransportWrapper) Send(request JSONRPCRequest) error {
	requestBytes, err := json.Marshal(request)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}
	
	if _, err := s.stdin.Write(append(requestBytes, '\n')); err != nil {
		return fmt.Errorf("failed to write request: %w", err)
	}
	return nil
}

func (s *stdioTransportWrapper) Receive() (*JSONRPCResponse, error) {
	// This should not be used directly for stdio
	return nil, fmt.Errorf("stdio transport uses async messaging")
}

func (s *stdioTransportWrapper) IsConnected() bool {
	return s.cmd != nil && s.cmd.Process != nil
}


func (c *MCPClient) initialize() error {
	c.setState(StateInitializing)
	
	request := JSONRPCRequest{
		JSONRpc: "2.0",
		ID:      c.nextRequestID(),
		Method:  "initialize",
		Params: InitializeRequest{
			ProtocolVersion: "2024-11-05",
			Capabilities: ClientCapabilities{
				Roots: &RootsCapability{
					ListChanged: true,
				},
			},
			ClientInfo: ClientInfo{
				Name:    "ocstack-mcp-client",
				Version: "1.0.0",
			},
		},
	}
	
	ctx, cancel := context.WithTimeout(c.ctx, c.config.Timeout)
	defer cancel()
	
	response, err := c.sendRequest(ctx, request)
	if err != nil {
		return fmt.Errorf("initialize request failed: %w", err)
	}
	
	if response.Error != nil {
		return fmt.Errorf("initialize failed: %s", response.Error.Message)
	}
	
	var initResponse InitializeResponse
	resultBytes, err := json.Marshal(response.Result)
	if err != nil {
		return fmt.Errorf("failed to marshal initialize response: %w", err)
	}
	
	if err := json.Unmarshal(resultBytes, &initResponse); err != nil {
		return fmt.Errorf("failed to unmarshal initialize response: %w", err)
	}
	
	c.mu.Lock()
	c.serverInfo = &initResponse.ServerInfo
	c.capabilities = &initResponse.Capabilities
	c.mu.Unlock()
	
	// Send initialized notification
	notification := JSONRPCRequest{
		JSONRpc: "2.0",
		Method:  "notifications/initialized",
	}
	
	return c.sendNotification(notification)
}

func (c *MCPClient) refreshTools() error {
	tools, err := c.ListTools(c.ctx)
	if err != nil {
		return err
	}
	
	c.mu.Lock()
	c.tools = tools
	c.mu.Unlock()
	
	return nil
}

func (c *MCPClient) handleMessages() {
	// Get stdout from the stdio transport wrapper
	var stdout io.ReadCloser
	if stdioWrapper, ok := c.transport.(*stdioTransportWrapper); ok {
		stdout = stdioWrapper.stdout
	} else {
		fmt.Printf("Error: handleMessages called on non-stdio transport\n")
		return
	}
	
	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}
		
		var response JSONRPCResponse
		if err := json.Unmarshal([]byte(line), &response); err != nil {
			fmt.Printf("Failed to parse JSON-RPC response: %v\n", err)
			continue
		}
		
		c.mu.RLock()
		ch, exists := c.responses[response.ID]
		c.mu.RUnlock()
		
		if exists {
			select {
			case ch <- response:
			case <-c.ctx.Done():
				return
			}
		}
	}
}

func (c *MCPClient) sendRequest(ctx context.Context, request JSONRPCRequest) (*JSONRPCResponse, error) {
	switch c.config.Transport {
	case TransportHTTP:
		// For HTTP, use synchronous request/response
		if httpTransport, ok := c.transport.(*HTTPTransport); ok {
			return httpTransport.SendRequest(ctx, request)
		}
		return nil, fmt.Errorf("HTTP transport not properly initialized")
		
	case TransportWebSocket, TransportStdio:
		// For WebSocket and stdio, use async messaging
		return c.sendAsyncRequest(ctx, request)
		
	default:
		return nil, fmt.Errorf("unsupported transport type: %s", c.config.Transport)
	}
}

func (c *MCPClient) sendAsyncRequest(ctx context.Context, request JSONRPCRequest) (*JSONRPCResponse, error) {
	// Create response channel
	responseCh := make(chan JSONRPCResponse, 1)
	c.mu.Lock()
	c.responses[request.ID] = responseCh
	c.mu.Unlock()
	
	defer func() {
		c.mu.Lock()
		delete(c.responses, request.ID)
		c.mu.Unlock()
	}()
	
	// Send request
	if err := c.transport.Send(request); err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	
	// Wait for response
	select {
	case response := <-responseCh:
		return &response, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-c.ctx.Done():
		return nil, c.ctx.Err()
	}
}

func (c *MCPClient) sendNotification(notification JSONRPCRequest) error {
	switch c.config.Transport {
	case TransportHTTP:
		// For HTTP, we need to use the synchronous method
		if httpTransport, ok := c.transport.(*HTTPTransport); ok {
			ctx, cancel := context.WithTimeout(c.ctx, c.config.Timeout)
			defer cancel()
			_, err := httpTransport.SendRequest(ctx, notification)
			return err
		}
		return fmt.Errorf("HTTP transport not properly initialized")
	default:
		// For other transports, use the async method
		if err := c.transport.Send(notification); err != nil {
			return fmt.Errorf("failed to send notification: %w", err)
		}
		return nil
	}
}

func (c *MCPClient) nextRequestID() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.requestID++
	return c.requestID
}

func (c *MCPClient) convertSchema(schema ToolSchema) *Parameters {
	params := &Parameters{
		Type:       schema.Type,
		Required:   schema.Required,
		Properties: make(map[string]*Properties),
	}
	
	for name, prop := range schema.Properties {
		if propMap, ok := prop.(map[string]interface{}); ok {
			property := &Properties{}
			if propType, exists := propMap["type"].(string); exists {
				property.Type = propType
			}
			if description, exists := propMap["description"].(string); exists {
				property.Description = description
			}
			if enum, exists := propMap["enum"].([]interface{}); exists {
				for _, e := range enum {
					if enumStr, ok := e.(string); ok {
						property.Enum = append(property.Enum, enumStr)
					}
				}
			}
			params.Properties[name] = property
		}
	}
	
	return params
}