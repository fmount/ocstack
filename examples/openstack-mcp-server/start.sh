#!/bin/bash

# OpenStack MCP Server Startup Script (using uv)

echo "Starting OpenStack MCP HTTP Server..."
echo "======================================="

# Check for uv
if ! command -v uv &> /dev/null; then
    echo "Error: 'uv' is not installed or not found in PATH"
    echo "Please install uv first:"
    echo "  curl -LsSf https://astral.sh/uv/install.sh | sh"
    echo "  or visit: https://github.com/astral-sh/uv"
    echo ""
    exit 1
fi

# Check for KUBECONFIG
if [ -z "$KUBECONFIG" ]; then
    echo "Warning: KUBECONFIG environment variable is not set"
    echo "Please set it before starting the server:"
    echo "  export KUBECONFIG=/path/to/your/kubeconfig"
    echo ""
fi

# Check for oc command
if ! command -v oc &> /dev/null; then
    echo "Warning: 'oc' command not found in PATH"
    echo "Please ensure OpenShift CLI is installed and accessible"
    echo ""
fi

# Set default values
export MCP_HOST=${MCP_HOST:-localhost}
export MCP_PORT=${MCP_PORT:-8080}
export DEFAULT_NAMESPACE=${DEFAULT_NAMESPACE:-openstack}

echo "Configuration:"
echo "  Host: $MCP_HOST"
echo "  Port: $MCP_PORT"
echo "  Default Namespace: $DEFAULT_NAMESPACE"
echo "  KUBECONFIG: ${KUBECONFIG:-'Not set'}"
echo "  uv version: $(uv --version)"
echo ""

# Install dependencies with uv (no sync needed, just install)
echo "Installing dependencies with uv..."
uv pip install fastapi uvicorn pydantic requests

echo ""
echo "Starting server at http://$MCP_HOST:$MCP_PORT"
echo "MCP endpoint: http://$MCP_HOST:$MCP_PORT/mcp"
echo "Health check: http://$MCP_HOST:$MCP_PORT/health"
echo ""
echo "Press Ctrl+C to stop the server"
echo ""

# Start the server using uv
uv run --with fastapi --with uvicorn --with pydantic --with requests server.py