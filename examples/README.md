# Examples

This directory contains example MCP (Model Context Protocol) servers that can be used with the ocstack project.

## openstack-mcp-server

A complete example of an HTTP-based MCP server that exposes OpenStack management tools. This server demonstrates:

- **MCP Protocol Implementation**: Full JSON-RPC 2.0 over HTTP implementation
- **Tool Integration**: All OpenStack management functions from ocstack
- **HTTP API**: RESTful endpoints for MCP protocol communication
- **FastAPI Framework**: Modern Python web framework implementation

### Usage

1. Navigate to the server directory:
   ```bash
   cd examples/openstack-mcp-server
   ```

2. Install dependencies:
   ```bash
   python -m venv venv
   source venv/bin/activate
   pip install -r requirements.txt
   ```

3. Start the server:
   ```bash
   python server.py
   ```

4. Connect from ocstack:
   ```bash
   # In the main ocstack project directory
   go run .
   
   # Connect to the MCP server
   Q :> /mcp connect http http://localhost:8080/mcp
   
   # Use the tools
   Q :> What is the current OpenStack control plane status?
   ```

### Files

- `server.py` - Main FastAPI server implementing MCP protocol
- `mcp_types.py` - MCP protocol type definitions and schemas  
- `openstack_tools.py` - OpenStack tool implementations
- `pyproject.toml` - Python project configuration
- `requirements.txt` - Python dependencies
- `start.sh` - Convenience startup script
- `test_client.py` - Test client for development

This example serves as both a working MCP server and a reference implementation for creating your own MCP servers to extend ocstack's capabilities.