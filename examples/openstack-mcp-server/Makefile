# OpenStack MCP Server Makefile

.PHONY: help install run test clean dev lint format

help: ## Show this help message
	@echo "OpenStack MCP Server"
	@echo "==================="
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-15s\033[0m %s\n", $$1, $$2}'

install: ## Install dependencies with uv
	uv sync

run: ## Start the MCP server
	./start.sh

dev: ## Install dev dependencies  
	uv sync --dev

test: ## Run tests with uv
	uv run test_client.py

lint: ## Lint code with ruff
	uv run ruff check .

format: ## Format code with black
	uv run black .

clean: ## Clean up generated files
	rm -rf .pytest_cache/
	rm -rf __pycache__/
	find . -name "*.pyc" -delete
	find . -name "*.pyo" -delete

check: lint ## Run code quality checks
	uv run ruff check .
	@echo "✅ Code quality checks passed"

server: ## Start server directly with uv
	uv run server.py

client: ## Run test client  
	uv run test_client.py

health: ## Check server health
	curl -s http://localhost:8080/health | python -m json.tool

tools: ## List available tools (debug)
	curl -s http://localhost:8080/tools | python -m json.tool

# Development targets
init: ## Initialize new uv project
	uv init --no-readme --name openstack-mcp-server

deps: ## Add new dependencies
	uv add $(PACKAGE)

dev-deps: ## Add new dev dependencies  
	uv add --dev $(PACKAGE)

# Example: make deps PACKAGE="requests"
# Example: make dev-deps PACKAGE="pytest"