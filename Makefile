# Variables
APP_NAME := ocstack
PKG := ./...

# Build the binary
build:
	go build -o bin/$(APP_NAME) main.go

# Run the app
run:
	go run main.go

# Run tests
test:
	go test -v $(PKG)

# Clean build artifacts
clean:
	rm -rf bin

# Format code
fmt:
	go fmt $(PKG)

vet:
	go vet $(PKG)

# Lint (optional, requires golangci-lint or similar tool)
lint: fmt vet
	golangci-lint run

tidy:
	go mod tidy

# MCP Server targets
mcp-server-deps:
	@echo "Installing MCP server dependencies..."
	@cd examples/openstack-mcp-server && UV_VENV_CLEAR=0 uv venv
	@cd examples/openstack-mcp-server && uv pip install -r requirements.txt
	@echo "MCP server dependencies installed"

mcp-server: mcp-server-deps
	@echo "Starting OpenStack MCP server at http://localhost:8080/mcp"
	@echo "Press Ctrl+C to stop the server"
	@cd examples/openstack-mcp-server && ./start.sh

mcp-server-stop:
	@echo "Stopping OpenStack MCP server..."
	@pkill -f "server.py" || echo "No MCP server process found"
	@echo "MCP server stopped"

.PHONY: build run test clean fmt lint mcp-server-deps mcp-server mcp-server-stop
