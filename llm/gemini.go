package llm


import (
    "context"
    "encoding/json"
    "fmt"
    "log"
    "strings"
    "google.golang.org/genai"

    "github.com/fmount/ocstack/pkg/ocstack"
    "github.com/fmount/ocstack/tools"
)

const (
	MODEL = "gemini-2.5-flash"
)

type GeminiProvider struct {
	client genai.Client
}

// GenerateChat -
func (c *GeminiProvider) GenerateChat(
	ctx context.Context,
	input string,
	s *Session,
) error {

	// If it's the first message, let's set some context in the history
	if len(s.GetHistory().Text) == 0 {
		s.UpdateContext()
	}

	// Convert tools to Gemini function declarations (but not during collective processing)
	var tools []*genai.Tool
	if len(s.Tools) > 0 && !s.ProcessingCollective {
		funcDeclarations, err := c.ConvertToGeminiFunctions(s.Tools)
		if err == nil {
			tools = []*genai.Tool{{FunctionDeclarations: funcDeclarations}}
		}
	}

	// Build conversation history for Gemini
	var contents []*genai.Content
	
	// Add previous messages from history
	h := s.GetHistory()
	for _, item := range h.Text {
		if msg, ok := item.Text.(string); ok && msg != "" {
			role := "user"
			switch item.Role {
			case "assistant":
				role = "model"
			case "system":
				role = "user" // Gemini doesn't have system role, convert to user
			}
			content := &genai.Content{
				Parts: []*genai.Part{genai.NewPartFromText(msg)},
				Role: role,
			}
			contents = append(contents, content)
		}
	}

	// Add current user input
	userContent := &genai.Content{
		Parts: []*genai.Part{genai.NewPartFromText(input)},
		Role: "user",
	}
	contents = append(contents, userContent)

	// Generate content
	config := &genai.GenerateContentConfig{
		Tools: tools,
	}
	
	// Enable function calling if tools are available
	if len(tools) > 0 {
		config.ToolConfig = &genai.ToolConfig{
			FunctionCallingConfig: &genai.FunctionCallingConfig{
				Mode: genai.FunctionCallingConfigModeAuto,
			},
		}
	}
	resp, err := c.client.Models.GenerateContent(ctx, MODEL, contents, config)
	if err != nil {
		return fmt.Errorf("failed to generate content: %v", err)
	}

	if resp == nil {
		return fmt.Errorf("received nil response from Gemini")
	}

	
	if resp != nil && len(resp.Candidates) > 0 {
		candidate := resp.Candidates[0]
		
		if candidate.Content != nil && len(candidate.Content.Parts) > 0 {
			var lastLLMResponse string
			var hadFunctionCalls bool
			var functionCalls []*genai.FunctionCall
			
			fmt.Printf("A :> ")
			
			// Process all parts of the response
			for _, part := range candidate.Content.Parts {
				
				if part.Text != "" {
					result := part.Text
					fmt.Printf("%s", result)
					lastLLMResponse += result
				}
				
				// Collect function calls for collective processing (like Ollama)
				if part.FunctionCall != nil {
					hadFunctionCalls = true
					functionCalls = append(functionCalls, part.FunctionCall)
				}
			}
			
			// If we have function calls, execute them collectively like Ollama does
			if len(functionCalls) > 0 {
				err := c.executeCollectiveFunctionCalls(ctx, s, functionCalls)
				if err != nil {
					fmt.Printf("T :> Function execution failed: %v\n", err)
				}
			}
			
			
			fmt.Printf("\n")
			
			// Update session history only if we have text response
			// Function calls will update history separately with their analysis
			if lastLLMResponse != "" {
				s.UpdateHistory(Message{
					Role: "assistant",
					Text: lastLLMResponse,
				})
			} else {
				// No text response - store function execution summary if we had function calls
				if hadFunctionCalls {
					summary := c.getFunctionCallSummary(functionCalls)
					s.UpdateHistory(Message{
						Role: "assistant", 
						Text: summary,
					})
				} else {
					// No text and no function calls - store a generic response
					s.UpdateHistory(Message{
						Role: "assistant",
						Text: "I understand.",
					})
				}
			}

			// Check for recommendations in LLM response
			CheckForRecommendations(s, lastLLMResponse)
		}
	}

	return nil
}

// ConvertToGeminiFunctions converts tools.Tool to genai.FunctionDeclaration (exported for testing)
func (c *GeminiProvider) ConvertToGeminiFunctions(toolsBytes []byte) ([]*genai.FunctionDeclaration, error) {
	var toolsList []tools.Tool
	err := json.Unmarshal(toolsBytes, &toolsList)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal tools: %v", err)
	}

	var funcDeclarations []*genai.FunctionDeclaration
	
	for _, tool := range toolsList {
		if tool.Function == nil {
			continue
		}
		
		// Convert parameters
		var schema *genai.Schema
		if tool.Function.Parameters != nil {
			schema = &genai.Schema{
				Type: genai.TypeObject,
				Properties: make(map[string]*genai.Schema),
				Required: tool.Function.Parameters.Required,
			}
			
			// Convert properties
			if tool.Function.Parameters.Properties != nil {
				for propName, prop := range tool.Function.Parameters.Properties {
					propSchema := &genai.Schema{
						Type: genai.TypeString, // Default to string, could be enhanced
						Description: prop.Description,
					}
					
					// Convert type if specified
					switch prop.Type {
					case "integer":
						propSchema.Type = genai.TypeInteger
					case "number":
						propSchema.Type = genai.TypeNumber
					case "boolean":
						propSchema.Type = genai.TypeBoolean
					case "array":
						propSchema.Type = genai.TypeArray
					case "object":
						propSchema.Type = genai.TypeObject
					}
					
					schema.Properties[propName] = propSchema
				}
			}
		}

		funcDecl := &genai.FunctionDeclaration{
			Name: tool.Function.Name,
			Description: tool.Function.Description,
			Parameters: schema,
		}
		
		funcDeclarations = append(funcDeclarations, funcDecl)
	}
	
	return funcDeclarations, nil
}

// executeCollectiveFunctionCalls processes all function calls collectively like Ollama does
func (c *GeminiProvider) executeCollectiveFunctionCalls(ctx context.Context, s *Session, functionCalls []*genai.FunctionCall) error {
	if len(functionCalls) == 0 {
		return nil
	}
	
	var toolResults []*tools.FunctionCall
	ns := s.GetConfig()[ocstack.NAMESPACE]
	
	// Execute all function calls and collect results
	for _, funcCall := range functionCalls {
		// Convert function call to tools.FunctionCall
		argsBytes, err := json.Marshal(funcCall.Args)
		if err != nil {
			continue
		}
		
		f, err := tools.ToFunctionCall(funcCall.Name, argsBytes)
		if err != nil {
			continue
		}
		
		var toolResult string
		
		// Only use MCP tools - no local tool fallback
		if mcpRegistry := s.GetMCPRegistry(); mcpRegistry != nil && mcpRegistry.IsToolFromMCP(f.Name) {
			// ALWAYS override namespace parameter with ocstack's configured
			// namespace
			if f.Arguments == nil {
				f.Arguments = make(map[string]any)
			}
			f.Arguments["namespace"] = ns
			// Execute MCP tool
			toolResult = mcpRegistry.ExecuteMCPTool(f)
			f.Result = toolResult
		} else if mcpRegistry == nil {
			toolResult = "MCP not connected. Use '/mcp connect' to enable tools."
			f.Result = toolResult
		} else {
			// Tool not available in MCP
			toolResult = fmt.Sprintf("Tool '%s' not available in MCP. Available tools can be seen with '/mcp tools'", f.Name)
			f.Result = toolResult
		}
		
		if s.Debug {
			fmt.Printf("[DEBUG] |-->> %s\n", f.Name)
			fmt.Printf("[DEBUG] | -->> out: %s\n", f.Result)
		}
		
		// Add to collection for collective analysis
		toolResults = append(toolResults, f)
	}
	
	// Process all tool results collectively using the template (like Ollama does)
	if len(toolResults) > 0 {
		collectivePrompt := tools.RenderCollectiveExec(toolResults)
		
		// Set flag to indicate collective processing (to prevent infinite loops)
		s.ProcessingCollective = true
		
		// Make recursive call to GenerateChat for collective analysis
		err := c.GenerateChat(ctx, collectivePrompt, s)
		
		// Reset flag
		s.ProcessingCollective = false
		
		if err != nil {
			return fmt.Errorf("collective analysis failed: %v", err)
		}
	}
	
	return nil
}

// getFunctionCallSummary creates a summary of executed function calls for history
func (c *GeminiProvider) getFunctionCallSummary(functionCalls []*genai.FunctionCall) string {
	if len(functionCalls) == 0 {
		return "Executed function calls"
	}
	
	if len(functionCalls) == 1 {
		return fmt.Sprintf("Executed function: %s", functionCalls[0].Name)
	}
	
	var names []string
	for _, fc := range functionCalls {
		names = append(names, fc.Name)
	}
	
	return fmt.Sprintf("Executed functions: %s", strings.Join(names, ", "))
}

// GetLLMClient - implements the interface defined in provider.go
func (p *GeminiProvider) GetLLMClient(ctx context.Context) (Client, error) {
    // The client gets the API key from the environment variable `GEMINI_API_KEY`.
    client, err := genai.NewClient(ctx, nil)
	if err != nil {
		log.Println(err)
		return nil, err
	}
	p.client = *client
	return p, nil
}
