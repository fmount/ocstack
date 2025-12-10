package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/fmount/ocstack/llm"
	"github.com/fmount/ocstack/mcp"
	"github.com/fmount/ocstack/pkg/ocstack"
	t "github.com/fmount/ocstack/template"
	tools "github.com/fmount/ocstack/tools"
)

const (
	DEBUG = true // switch to true to print additional information
)

// CliCommand -
func CliCommand(q string, s *llm.Session) {
	query := strings.ToLower(q)
	tokens := strings.Split(query, " ")
	tq := tokens[0]
	// tokenize and get the first item. Next items are passed as parameters to
	// the selected case
	switch {
	case tq == "exit" || tq == "quit":
		fmt.Println("Bye!")
		// TODO: dump sessions if any
		os.Exit(0)
	case tq == "read":
		fmt.Println("TODO: Read input from workspace path")
		// TODO:
		// - workspace is a path where we have assets that can be used as input
		// - workspace path can be set via an ENV variable
	case tq == "template":
		if len(tokens) < 2 {
			ocstack.TermHelper(tq)
			return
		}
		// no session, return
		if s == nil {
			ocstack.ShowWarn("No session")
			return
		}
		profile, err := t.LoadProfile(tokens[1])
		if err != nil {
			ocstack.ShowWarn(fmt.Sprintf("%s\n", err))
			return
		}
		ocstack.TermHeader(tokens[1])
		s.Profile = profile
		s.UpdateContext()
	case tq == "namespace":
		// set or update namespace
		s.SetConfig(ocstack.NAMESPACE, tokens[1])
	case tq == "config":
		// show config options
		s.ShowConfig()
	case tq == "mcp":
		// MCP connection commands
		if len(tokens) < 2 {
			fmt.Println("Usage: /mcp connect <command> | /mcp disconnect | /mcp tools")
			return
		}
		if s == nil {
			ocstack.ShowWarn("No session")
			return
		}
		switch tokens[1] {
		case "connect":
			if len(tokens) < 3 {
				fmt.Println("Usage: /mcp connect <server-type> [url]")
				fmt.Println("Available servers: filesystem, brave-search, sqlite, http, websocket")
				fmt.Println("For http/websocket, provide URL as third parameter")
				return
			}
			var url string
			if len(tokens) > 3 {
				url = tokens[3]
			}
			connectMCP(s, tokens[2], url)
		case "disconnect":
			disconnectMCP(s)
		case "tools":
			listMCPTools(s)
		default:
			fmt.Println("Unknown MCP command. Use: connect, disconnect, or tools")
		}
	case tq == "help":
		ocstack.TermHelper("")
		return
	default:
		fmt.Println("Default!")
		return
	}
}

// MCP helper functions
func connectMCP(s *llm.Session, serverType string, url string) {
	fmt.Printf("Connecting to MCP server: %s...\n", serverType)
	
	var config mcp.MCPConfig
	switch serverType {
	case "filesystem":
		config = mcp.FilesystemMCPConfig
	case "brave-search":
		config = mcp.BraveSearchMCPConfig
		fmt.Println("Note: Set BRAVE_API_KEY environment variable for brave-search")
	case "sqlite":
		config = mcp.SQLiteMCPConfig
	case "http":
		if url == "" {
			fmt.Println("Error: URL required for HTTP connection")
			fmt.Println("Usage: /mcp connect http <url>")
			return
		}
		config = mcp.MCPConfig{
			Transport: mcp.TransportHTTP,
			ServerURL: url,
		}
	case "websocket":
		if url == "" {
			fmt.Println("Error: URL required for WebSocket connection")
			fmt.Println("Usage: /mcp connect websocket <url>")
			return
		}
		config = mcp.MCPConfig{
			Transport: mcp.TransportWebSocket,
			ServerURL: url,
		}
	default:
		fmt.Printf("Unknown server type: %s\n", serverType)
		fmt.Println("Available types: filesystem, brave-search, sqlite, http, websocket")
		return
	}
	
	// Create MCP client
	client := mcp.NewClient(config)
	
	// Connect
	ctx := context.Background()
	if err := client.Connect(ctx); err != nil {
		fmt.Printf("Failed to connect to MCP server: %v\n", err)
		return
	}
	
	// Create MCP tool registry
	registry := mcp.NewMCPToolRegistry()
	registry.SetMCPClient(client)
	
	// Get local tools and add them to the registry
	localTools, err := tools.RegisterTools()
	if err != nil {
		fmt.Printf("Warning: failed to load local tools: %v\n", err)
	} else {
		registry.SetLocalTools(localTools)
	}
	
	// Update session with combined tools (MCP tools take priority)
	s.Tools = registry.GetAllTools()
	s.SetMCPRegistry(registry)
	
	fmt.Printf("Successfully connected to MCP server: %s\n", serverType)
}

func disconnectMCP(s *llm.Session) {
	if mcpRegistry := s.GetMCPRegistry(); mcpRegistry != nil {
		fmt.Println("Disconnecting MCP client...")
		// Fall back to local tools only
		localTools, err := tools.RegisterTools()
		if err != nil {
			fmt.Printf("Warning: failed to load local tools: %v\n", err)
			s.Tools = []byte("[]") // No tools available
		} else {
			s.Tools = localTools
		}
		s.SetMCPRegistry(nil)
		fmt.Println("Disconnected from MCP server - using local tools only")
	} else {
		fmt.Println("No MCP connection active")
	}
}

func listMCPTools(s *llm.Session) {
	if mcpRegistry := s.GetMCPRegistry(); mcpRegistry != nil {
		fmt.Println("Available tools (MCP + local, MCP takes priority):")
		
		// Get all tools from the registry
		allToolsData := mcpRegistry.GetAllTools()
		
		// Parse and display the tools
		var tools []map[string]interface{}
		if err := json.Unmarshal(allToolsData, &tools); err == nil {
			fmt.Printf("Found %d tools:\n\n", len(tools))
			
			for i, tool := range tools {
				if toolFunc, exists := tool["function"].(map[string]interface{}); exists {
					name := toolFunc["name"]
					description := toolFunc["description"]
					fmt.Printf("%d. %s\n", i+1, name)
					fmt.Printf("   Description: %s\n", description)
					
					if params, exists := toolFunc["parameters"].(map[string]interface{}); exists {
						if props, exists := params["properties"].(map[string]interface{}); exists && len(props) > 0 {
							fmt.Printf("   Parameters: ")
							var paramNames []string
							for paramName := range props {
								paramNames = append(paramNames, paramName)
							}
							fmt.Printf("%s\n", strings.Join(paramNames, ", "))
						}
					}
					fmt.Println()
				}
			}
		} else {
			fmt.Printf("Error parsing tools: %v\n", err)
			fmt.Printf("Raw tools data: %s\n", string(allToolsData))
		}
	} else {
		fmt.Println("No MCP connection active. Local tools only.")
		fmt.Println("Use '/mcp connect <server-type>' to connect to an MCP server")
	}
}

func main() {

	// Validate ocstack input required to access Tools
	tools.ExitOnErrors()

	ctx := context.Background()

	client, err := llm.GetProvider(llm.OLLAMAPROVIDER)
	if err != nil {
		panic(err)
	}

	h := llm.History{}
	// Register local tools - these will be available alongside MCP tools
	b, err := tools.RegisterTools()
	if err != nil {
		panic(err)
	}

	profile, err := t.LoadProfile("default")
	if err != nil {
		ocstack.ShowWarn(fmt.Sprintf("%s\n", err))
	}

	config := tools.LoadDefaultConfig()

	// Create a new session for the current execution before entering the
	// loop
	s, _ := llm.NewSession(
		llm.QWEN,
		profile,
		h,
		b,
		DEBUG,
		config,
	)

	// pass the loaded profile
	ocstack.TermHeader("default")

	for {
		fmt.Printf("Q :> ")
		// Read input
		reader := bufio.NewReader(os.Stdin)
		input, err := reader.ReadString('\n')
		if err != nil {
			log.Fatal(err)
		}

		// no input provided, go back to the beginning
		if len(strings.TrimSuffix(input, "\n")) == 0 {
			continue
		}

		// process potential commands
		if len(input) > 0 && strings.HasPrefix(input, "/") {
			// Trim any whitespace from the input
			q := strings.TrimSpace(input)
			CliCommand(strings.TrimPrefix(q, "/"), s)
			continue
		}

		// propagate the request to the LLM
		err = client.GenerateChat(
			ctx,
			input,
			s,
		)
		if s.Debug {
			fmt.Printf("[HISTORY]:\n")
			fmt.Println(s.GetHistory())
			fmt.Printf("----------\n")
		}
	}
}
