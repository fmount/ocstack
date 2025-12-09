package mcp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/gorilla/websocket"
)

// Transport defines the interface for different MCP communication methods
type Transport interface {
	Connect(ctx context.Context) error
	Disconnect() error
	Send(request JSONRPCRequest) error
	Receive() (*JSONRPCResponse, error)
	IsConnected() bool
}

// TransportType defines the type of transport
type TransportType string

const (
	TransportStdio     TransportType = "stdio"
	TransportHTTP      TransportType = "http"
	TransportWebSocket TransportType = "websocket"
)

// HTTPTransport implements MCP over HTTP
type HTTPTransport struct {
	baseURL    string
	httpClient *http.Client
	headers    map[string]string
	connected  bool
}

// NewHTTPTransport creates a new HTTP transport
func NewHTTPTransport(baseURL string, timeout time.Duration) *HTTPTransport {
	if timeout == 0 {
		timeout = 30 * time.Second
	}

	return &HTTPTransport{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: timeout,
		},
		headers: map[string]string{
			"Content-Type": "application/json",
			"Accept":       "application/json",
		},
	}
}

func (h *HTTPTransport) Connect(ctx context.Context) error {
	// For HTTP, we don't need a persistent connection, just validate the URL
	_, err := url.Parse(h.baseURL)
	if err != nil {
		return fmt.Errorf("invalid HTTP URL: %w", err)
	}
	h.connected = true
	return nil
}

func (h *HTTPTransport) Disconnect() error {
	h.connected = false
	return nil
}

func (h *HTTPTransport) IsConnected() bool {
	return h.connected
}

func (h *HTTPTransport) Send(request JSONRPCRequest) error {
	return fmt.Errorf("HTTP transport requires synchronous send/receive")
}

func (h *HTTPTransport) Receive() (*JSONRPCResponse, error) {
	return nil, fmt.Errorf("HTTP transport requires synchronous send/receive")
}

// SendRequest sends a request and returns the response for HTTP transport
func (h *HTTPTransport) SendRequest(ctx context.Context, request JSONRPCRequest) (*JSONRPCResponse, error) {
	if !h.connected {
		return nil, fmt.Errorf("HTTP transport not connected")
	}

	requestBytes, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", h.baseURL, bytes.NewReader(requestBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}

	// Set headers
	for key, value := range h.headers {
		req.Header.Set(key, value)
	}

	resp, err := h.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP request failed with status: %d", resp.StatusCode)
	}

	responseBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var response JSONRPCResponse
	if err := json.Unmarshal(responseBytes, &response); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &response, nil
}

// WebSocketTransport implements MCP over WebSocket
type WebSocketTransport struct {
	url        string
	conn       *websocket.Conn
	connected  bool
	sendCh     chan JSONRPCRequest
	receiveCh  chan JSONRPCResponse
	closeCh    chan struct{}
	dialer     *websocket.Dialer
}

// NewWebSocketTransport creates a new WebSocket transport
func NewWebSocketTransport(url string) *WebSocketTransport {
	return &WebSocketTransport{
		url: url,
		dialer: &websocket.Dialer{
			HandshakeTimeout: 30 * time.Second,
		},
		sendCh:    make(chan JSONRPCRequest, 10),
		receiveCh: make(chan JSONRPCResponse, 10),
		closeCh:   make(chan struct{}),
	}
}

func (w *WebSocketTransport) Connect(ctx context.Context) error {
	conn, _, err := w.dialer.DialContext(ctx, w.url, nil)
	if err != nil {
		return fmt.Errorf("failed to connect to WebSocket: %w", err)
	}

	w.conn = conn
	w.connected = true

	// Start message handling goroutines
	go w.sendLoop()
	go w.receiveLoop()

	return nil
}

func (w *WebSocketTransport) Disconnect() error {
	if !w.connected {
		return nil
	}

	w.connected = false
	close(w.closeCh)

	if w.conn != nil {
		w.conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
		w.conn.Close()
	}

	return nil
}

func (w *WebSocketTransport) IsConnected() bool {
	return w.connected
}

func (w *WebSocketTransport) Send(request JSONRPCRequest) error {
	if !w.connected {
		return fmt.Errorf("WebSocket not connected")
	}

	select {
	case w.sendCh <- request:
		return nil
	case <-w.closeCh:
		return fmt.Errorf("WebSocket transport closed")
	case <-time.After(5 * time.Second):
		return fmt.Errorf("send timeout")
	}
}

func (w *WebSocketTransport) Receive() (*JSONRPCResponse, error) {
	if !w.connected {
		return nil, fmt.Errorf("WebSocket not connected")
	}

	select {
	case response := <-w.receiveCh:
		return &response, nil
	case <-w.closeCh:
		return nil, fmt.Errorf("WebSocket transport closed")
	case <-time.After(30 * time.Second):
		return nil, fmt.Errorf("receive timeout")
	}
}

func (w *WebSocketTransport) sendLoop() {
	for {
		select {
		case request := <-w.sendCh:
			if err := w.conn.WriteJSON(request); err != nil {
				fmt.Printf("WebSocket send error: %v\n", err)
				w.Disconnect()
				return
			}
		case <-w.closeCh:
			return
		}
	}
}

func (w *WebSocketTransport) receiveLoop() {
	for {
		select {
		case <-w.closeCh:
			return
		default:
			var response JSONRPCResponse
			if err := w.conn.ReadJSON(&response); err != nil {
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
					fmt.Printf("WebSocket receive error: %v\n", err)
				}
				w.Disconnect()
				return
			}

			select {
			case w.receiveCh <- response:
			case <-w.closeCh:
				return
			}
		}
	}
}

// StdioTransport remains the same as before but implements the Transport interface
type StdioTransport struct {
	cmd    interface{} // Will be *exec.Cmd
	stdin  io.WriteCloser
	stdout io.ReadCloser
	stderr io.ReadCloser
	sendCh chan JSONRPCRequest
	recvCh chan JSONRPCResponse
	ctx    context.Context
	cancel context.CancelFunc
}

// Placeholder implementation - the actual stdio implementation should be moved here
func (s *StdioTransport) Connect(ctx context.Context) error {
	// Implementation would be moved from the client
	return nil
}

func (s *StdioTransport) Disconnect() error {
	return nil
}

func (s *StdioTransport) Send(request JSONRPCRequest) error {
	return nil
}

func (s *StdioTransport) Receive() (*JSONRPCResponse, error) {
	return nil, nil
}

func (s *StdioTransport) IsConnected() bool {
	return false
}