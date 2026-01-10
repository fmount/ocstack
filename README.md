# ocstack – Your AI Assistant for OpenStack on OpenShift

ocstack is an experimental AI-powered assistant designed to simplify and
automate common tasks in OpenStack environments running on OpenShift. This tool
serves as a proof of concept (PoC) to explore the capabilities of local large
language models (LLMs) through the Ollama framework, using intelligent agents
to interact with complex cloud-native platforms.

## Why

Managing OpenStack on Kubernetes (specifically OpenShift) often involves a
steep learning curve, complex CLI interactions, and context switching across
multiple tools. ocstack leverages LLM-powered agents to:

- Provide conversational interaction with your OpenStack environment
- Guide users through operational tasks (e.g., deployment checks, status reports)
- Reduce cognitive load for new users or operators
- Enable experimentation with local LLMs in secure, air-gapped environments

By integrating LLM agents into your cloud workflows, ocstack demonstrates the
potential of intelligent automation in infrastructure management—where agents
can understand context, make suggestions, and execute commands with minimal
human input.

Unlike traditional scripts or hardcoded automation, agents can:

- Interpret natural language and convert it into actionable commands
- Adapt to varying environments and scenarios
- Offer explanations or troubleshooting steps when something goes wrong

In this PoC, agents act as a bridge between human operators and the
OpenStack/OpenShift ecosystem—empowering users to get help, ask questions, or
automate repetitive tasks in real-time.

**Demo:** https://asciinema.org/a/722296

## Chat with your OpenStack environment

Once set up, you can interact with your OpenStack deployment using natural
language prompts. Ask for node status, deployment logs, or even guidance on
common workflows—all through a conversational interface.

## Getting started

> Note: This project is in active development and subject to significant changes as new capabilities are tested and added.

### Prerequisites

Install the necessary tools and prepare your OpenShift (CRC) environment:

```bash
cd ~
$ git clone https://github.com/openstack-k8s-operators/install_yamls.git
$ cd install_yamls/devsetup
$ PULL_SECRET=~/pull-secret CPUS=6 MEMORY=20480 make download_tools crc
```

Follow the [install_yamls documentation]() to deploy an OpenStack environment
on OpenShift.

### Run the ocstack assistant

## Run OLLAMA

You can now run ocstack using a local Ollama server for offline LLM support,
instead of relying on remote APIs:

```bash
curl -fsSL https://ollama.com/install.sh | sh
```

### Download a model (e.g., qwen3):

```bash
ollama pull qwen3:latest
```

### Start the Ollama server:

```bash
ollama serve &
```

Alternatively, depending on the environment, use systemd:

```bash
systemctl start ollama
```

and watch it using journalctl.

This starts a local REST API compatible with OpenAI’s `v1/chat/completions` at
`http://127.0.0.1:11434`.

Once the openstack on openshift environment is ready, and `OLLAMA` serves a model,
start the ocstack assistant:

```bash
$ export KUBECONFIG=$HOME/.crc/machines/crc/kubeconfig; make build && make run
```

> **Note:** You can point to **any OpenShift environment** by updating the `KUBECONFIG` path to your desired cluster configuration.


## Ramalama Support (LLama.cpp via HTTP)

**ocstack** includes basic support for models served via
[ramalama.ai](https://ramalama.ai/) which provides a local runtime for LLMs
using LLama.cpp-compatible APIs. This allows the assistant to run fully offline
and self-hosted, which is ideal for development or to conduct experiments.

> **Note:** While chat interaction works, **function/tool calling is not yet
> supported** with the `LLAMACPP` provider in ocstack. To use tools, connect to an MCP server, but the LlamaCpp provider currently cannot invoke them.

### How to Use ocstack with Ramalama

#### **Start Ramalama**

Install and start a model using the istructions provided by
[ramalama.ai](https://ramalama.ai/):

```bash
ramalama serve llama3
```

This will expose an API compatible with OpenAI's `v1/chat/completions`,
typically on `http://localhost:8080`.

#### **Export the LLM Endpoint**

Set the environment variable so `ocstack` can locate your local ramalama
server:

```bash
export LLAMA_HOST=http://localhost:8080
```

#### **Switch the LLM Provider to LLAMACPP**

In your `main.go`, update the provider selection to use `LLAMACPP`:

```diff
- client, err := llm.GetProvider(llm.OLLAMAPROVIDER)
+ client, err := llm.GetProvider(llm.LLAMACPP)
```

> **Note:** This is still required because there's no cli yet as this is a
> very simple POC

#### 4. **Build and Run**

```bash
$ export KUBECONFIG=$HOME/.crc/machines/crc/kubeconfig; make build && make run
```

## Google Gemini Support

**ocstack** includes support for Google's Gemini models via the official Google Generative AI Go SDK. This provider supports both chat interaction and full function/tool calling capabilities.

### How to Use ocstack with Gemini

#### **Export the API Key**

Set the environment variable so `ocstack` can authenticate with Gemini:

```bash
export GEMINI_API_KEY=your_api_key_here
```

#### **Switch the LLM Provider to GEMINI**

In your `main.go`, update the provider selection to use `GEMINI`:

```diff
- client, err := llm.GetProvider(llm.OLLAMAPROVIDER)
+ client, err := llm.GetProvider(llm.GEMINI)
```

> **Note:** This change is temporary as this is still a PoC without CLI configuration

#### **Build and Run**

```bash
$ export KUBECONFIG=$HOME/.crc/machines/crc/kubeconfig; make build && make run
```

### Gemini Features

- **Full Tool Support**: Unlike the LLAMACPP provider, Gemini supports complete function/tool calling
- **MCP Integration**: Works seamlessly with MCP tools (local tools have been removed)
- **Advanced Reasoning**: Leverages Gemini 2.5 Flash model for intelligent responses and recommendations
- **Cloud-based**: No local model download required, but requires internet connection
- **Collective Processing**: Processes multiple tool calls together for comprehensive analysis

## Available Makefile Targets

OCStack provides convenient Makefile targets for building, running, and managing the MCP server:

### Core Targets

- `make build` - Build the ocstack binary
- `make run` - Run ocstack (requires build first)
- `make clean` - Clean build artifacts
- `make test` - Run tests
- `make fmt` - Format Go code
- `make lint` - Run linters (requires golangci-lint)

### MCP Server Targets

- `make mcp-server` - Start the OpenStack MCP server (includes dependency installation)
- `make mcp-server-deps` - Install MCP server dependencies only
- `make mcp-server-stop` - Stop the running MCP server

### Example Workflow

```bash
# Start MCP server in one terminal
make mcp-server

# In another terminal, build and run ocstack
export KUBECONFIG=$(pwd)/kubeconfig
make build && make run

# Connect to MCP and start using tools
Q :> /mcp connect http http://localhost:8080/mcp
Q :> What is the deployed OpenStack version in the 'openstack' namespace?
```

## MCP Server Integration

OCStack supports both **local tools** and **MCP tools** with a hybrid approach where MCP tools take priority when available.

### Running the Example MCP Server

OCStack includes a complete OpenStack MCP server example in `examples/openstack-mcp-server/`.

#### Quick Start

```bash
# Start the MCP server (requires Python 3.9+)
make mcp-server

# In another terminal, start ocstack
export KUBECONFIG=$(pwd)/kubeconfig  # Set your OpenShift config
make build && make run

# Connect to the MCP server
Q :> /mcp connect http http://localhost:8080/mcp

# List available tools
Q :> /mcp tools

# Use OpenStack tools via MCP
Q :> What is the deployed OpenStack version?
Q :> Check the status of Nova service
```

#### Manual Setup

If you prefer manual setup:

```bash
# Navigate to the MCP server directory
cd examples/openstack-mcp-server

# Create virtual environment and install dependencies
python3 -m venv venv
source venv/bin/activate
pip install -r requirements.txt

# Start the server
python server.py
```

### Available Tools

The MCP server provides these OpenStack management tools:

| Tool | Description | Parameters |
|------|-------------|------------|
| `hello` | Test function | `name` (string) |
| `oc` | Run OpenShift CLI commands | `command` (string) |
| `get_openstack_control_plane` | Get control plane status | `namespace` (optional) |
| `check_openstack_svc` | Check service status | `service` (required), `namespace` (optional) |
| `needs_minor_update` | Check if update needed | `namespace` (optional) |
| `get_deployed_version` | Get current version | `namespace` (optional) |
| `get_available_version` | Get available version | `namespace` (optional) |

**Note**: The local tools include one additional tool (`trigger_minor_update`) that's only available locally.

### MCP Commands

- `/mcp connect http http://localhost:8080/mcp` - Connect to HTTP MCP server
- `/mcp disconnect` - Disconnect and fall back to local tools
- `/mcp tools` - List all available tools (MCP + local)

### Configuration

The MCP server can be configured via environment variables:

```bash
export KUBECONFIG=/path/to/your/kubeconfig    # Required for OpenStack tools
export MCP_HOST=localhost                     # Server host (default: localhost)
export MCP_PORT=8080                         # Server port (default: 8080)
export DEFAULT_NAMESPACE=openstack          # Default 'openstack' namespace
```


### Troubleshooting

#### MCP Connection Issues

- **"client not connected"**: Ensure the MCP server is running (`make mcp-server`)
- **Connection timeouts**: Check if the server is accessible at `http://localhost:8080/health`
- **Tool execution hangs**: Verify KUBECONFIG is set and OpenShift cluster is accessible
- **Tool timeouts**: Check network connectivity to OpenShift cluster
- **Permission errors**: Verify your OpenShift user has proper RBAC permissions
