# OpenStack MCP HTTP Server

A standalone HTTP-based Model Context Protocol (MCP) server that exposes OpenStack management tools. This server implements the same functionality as the local tools in the ocstack project but makes them available via HTTP MCP protocol.

## Features

- **HTTP MCP Server**: RESTful API following MCP protocol specifications
- **OpenStack Tools**: All the OpenStack management functions from ocstack
- **Kubernetes Integration**: Uses `oc` commands for OpenStack operator management
- **JSON-RPC 2.0**: Standard MCP protocol over HTTP

## Tools Exposed

1. `hello` - Simple test function
2. `oc` - Run OpenShift client commands
3. `get_openstack_control_plane` - Get OpenStack control plane information
4. `check_openstack_svc` - Check OpenStack service status
5. `needs_minor_update` - Check if minor updates are needed
6. `get_deployed_version` - Get currently deployed OpenStack version
7. `get_available_version` - Get available OpenStack version

## Installation

```bash
# Create virtual environment
python3 -m venv venv
source venv/bin/activate

# Install dependencies
pip install -r requirements.txt

# Run the server
python server.py
```

## Usage

The server runs on `http://localhost:8080/mcp` by default.

### Connect from ocstack client:

```bash
# In your ocstack project
go run .

# Connect to the HTTP MCP server
Q :> /mcp connect http http://localhost:8080/mcp

# List available tools
Q :> /mcp tools

# Use the tools
Q :> What is the current OpenStack control plane status?
```

## Configuration

Set the following environment variables:

- `KUBECONFIG`: Path to your OpenShift/Kubernetes config
- `MCP_HOST`: Server host (default: localhost)
- `MCP_PORT`: Server port (default: 8080)
- `DEFAULT_NAMESPACE`: Default OpenStack namespace (default: openstack)

## API Endpoints

- `POST /mcp` - Main MCP protocol endpoint
- `GET /health` - Health check
- `GET /tools` - List available tools (for debugging)