# MCP Client Integration

This project now includes an MCP (Model Context Protocol) client that allows you to connect to external MCP servers and use their tools alongside your local tools.

## Features

- **MCP Client**: Full MCP protocol implementation with JSON-RPC over stdio
- **Tool Integration**: Seamless integration with existing tool system
- **Multiple Servers**: Support for filesystem, brave-search, sqlite, and custom MCP servers
- **Session Management**: MCP tools are integrated into the session and can be used by the LLM

## Usage

### Basic Commands

1. **Connect to an MCP server**:
   ```
   /mcp connect filesystem
   ```

2. **List available tools**:
   ```
   /mcp tools
   ```

3. **Disconnect from MCP server**:
   ```
   /mcp disconnect
   ```

### Supported MCP Servers

**Local Servers (stdio transport):**
- **filesystem**: File system operations (requires Node.js)
- **brave-search**: Web search using Brave API (requires BRAVE_API_KEY)
- **sqlite**: SQLite database operations

**Remote Servers (HTTP/WebSocket transport):**
- **http**: Connect to HTTP-based MCP servers
- **websocket**: Connect to WebSocket-based MCP servers

### Example Sessions

**Local MCP Server:**
```
Q :> /mcp connect filesystem
Connecting to MCP server: filesystem...
Successfully connected to MCP server: filesystem

Q :> /mcp tools
Available tools (local + MCP):
- Tools are loaded and available for the LLM

Q :> List the files in /tmp directory
A :> [The LLM can now use MCP filesystem tools to list files]
```

**Remote HTTP MCP Server:**
```
Q :> /mcp connect http https://api.example.com/mcp
Connecting to MCP server: http...
Successfully connected to MCP server: http

Q :> /mcp tools
Available tools (local + MCP):
- Tools are loaded and available for the LLM

Q :> [Use remote MCP tools via HTTP]
```

**Remote WebSocket MCP Server:**
```
Q :> /mcp connect websocket wss://mcp.example.com/ws
Connecting to MCP server: websocket...
Successfully connected to MCP server: websocket
```

## Architecture

### MCP Module Structure

- `mcp/types.go`: MCP protocol types and JSON-RPC structures
- `mcp/client.go`: MCP client implementation with stdio communication
- `mcp/adapter.go`: Tool adapter and registry for integrating MCP tools

### Integration Points

1. **Session Integration**: `Session` struct now includes MCP registry support
2. **Tool Execution**: OLLAMA provider automatically routes tool calls to MCP when appropriate
3. **Tool Registration**: Combined local and MCP tools are available to the LLM

## Configuration

You can create custom MCP configurations by modifying the `MCPConfig` struct:

```go
customConfig := mcp.MCPConfig{
    Command: []string{"your-mcp-server", "arg1", "arg2"},
    Env: map[string]string{
        "API_KEY": "your-api-key",
    },
    Timeout: 30 * time.Second,
}
```

## Troubleshooting

1. **Connection Issues**: Ensure MCP server is installed and accessible
2. **Permission Issues**: Check file permissions for stdio communication
3. **Tool Not Found**: Verify MCP server is properly connected with `/mcp tools`
4. **Debug Mode**: Set `DEBUG=true` to see detailed tool execution information
