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

Once the openstack on openshift environment is ready, start the ocstack
assistant:

```bash
$ export KUBECONFIG=$HOME/.crc/machines/crc/kubeconfig; make build && make run
```

> **Note:** You can point to **any OpenShift environment** by updating the `KUBECONFIG` path to your desired cluster configuration.

## Working with Local Tools

OCStack currently supports only local tool execution. Support for connecting to
MCP (Model Context Protocol) endpoints is planned for a future release.

The tool system is designed to be extensible, allowing you to create
specialized tools for specific tasks. Currently, a basic set of tools is
provided, but you can easily add custom tools to enhance functionality.

### Adding Custom Tools

To add a new tool, follow these three steps:

#### 1. Define the Tool Schema

Create a JSON definition file in the `tools/local` directory. Use descriptive
names that reflect the tool's purpose.

**Example:** For OpenStack-specific tools, create `tools/local/openstack.json`:

```json
[
  {
    "type": "function",
    "function": {
      "name": "get_endpoint_list",
      "description": "Get the OpenStack endpoint list",
      "parameters": {
        "type": "object",
        "properties": {
          "namespace": {
            "type": "string",
            "description": "The namespace where the OpenStack client is deployed"
          }
        },
        "required": ["namespace"]
      }
    }
  }
]
```

> Tool definitions from custom files are automatically merged with the
default `tools.json`, making them available to the LLM.

#### 2. Implement the Tool Logic

Add your tool's implementation to the `tools/utils.go` module. This is where
you define the actual functions that will be executed when the LLM calls your
tool.

#### 3. Register the Tool Handler

Update the `GenerateChat` function to handle calls to your new tool. This
connects the LLM's tool invocation to your implementation.

### Tool Organization

- **Default tools:** Defined in the base `tools.json` file
- **Custom tools:** Organized by category in `tools/local/` (e.g., `openstack.json`, `kubernetes.json`)
- **Implementation:** All tool logic resides in `tools/utils.go`
- **Integration:** Tool handlers are registered in the `GenerateChat` function


## Ramalama Support (LLama.cpp via HTTP)

**ocstack** includes basic support for models served via
[ramalama.ai](https://ramalama.ai/) which provides a local runtime for LLMs
using LLama.cpp-compatible APIs. This allows the assistant to run fully offline
and self-hosted, which is ideal for development or to conduct experiments.

> **Note:** While chat interaction works, **function/tool calling is not yet
> supported** with the `LLAMACPP` provider in ocstack.

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
