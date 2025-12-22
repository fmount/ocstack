package llm


import (
    "context"
    "encoding/json"
    "fmt"
    "log"
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
	if s.Debug {
		fmt.Printf("[DEBUG] - Using Gemini model: %s\n", MODEL)
	}

	// If it's the first message, let's set some context in the history
	if len(s.GetHistory().Text) == 0 {
		s.UpdateContext()
	}

	// Convert tools to Gemini function declarations
	var tools []*genai.Tool
	if len(s.Tools) > 0 {
		funcDeclarations, err := c.convertToGeminiFunctions(s.Tools)
		if err != nil {
			if s.Debug {
				fmt.Printf("[DEBUG] - Error converting tools: %v\n", err)
			}
		} else {
			tools = []*genai.Tool{{FunctionDeclarations: funcDeclarations}}
			if s.Debug {
				fmt.Printf("[DEBUG] - Added %d function declarations\n", len(funcDeclarations))
			}
		}
	}

	// Build conversation history for Gemini
	var contents []*genai.Content
	
	// Add previous messages from history
	h := s.GetHistory()
	for _, item := range h.Text {
		if msg, ok := item.Text.(string); ok {
			role := "user"
			if item.Role == "assistant" {
				role = "model"
			} else if item.Role == "system" {
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
	resp, err := c.client.Models.GenerateContent(ctx, MODEL, contents, config)
	if err != nil {
		return fmt.Errorf("failed to generate content: %v", err)
	}

	// Process response
	if resp != nil && len(resp.Candidates) > 0 {
		candidate := resp.Candidates[0]
		if candidate.Content != nil && len(candidate.Content.Parts) > 0 {
			var lastLLMResponse string
			
			fmt.Printf("A :> ")
			
			// Process all parts of the response
			for _, part := range candidate.Content.Parts {
				if part.Text != "" {
					result := part.Text
					fmt.Printf("%s", result)
					lastLLMResponse += result
				}
				
				// Handle function calls
				if part.FunctionCall != nil {
					if s.Debug {
						fmt.Printf("\n[DEBUG] - Function call: %s\n", part.FunctionCall.Name)
					}
					
					// Execute the function call
					err := c.executeFunctionCall(ctx, s, part.FunctionCall, contents)
					if err != nil && s.Debug {
						fmt.Printf("[DEBUG] - Error executing function: %v\n", err)
					}
				}
			}
			
			fmt.Printf("\n")
			
			// Update session history
			s.UpdateHistory(Message{
				Role: "assistant",
				Text: lastLLMResponse,
			})

			// Check for recommendations in LLM response
			CheckForRecommendations(s, lastLLMResponse)
		}
	}

	return nil
}

// convertToGeminiFunctions converts tools.Tool to genai.FunctionDeclaration
func (c *GeminiProvider) convertToGeminiFunctions(toolsBytes []byte) ([]*genai.FunctionDeclaration, error) {
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

// executeFunctionCall executes a function call and continues the conversation
func (c *GeminiProvider) executeFunctionCall(ctx context.Context, s *Session, funcCall *genai.FunctionCall, contents []*genai.Content) error {
	if funcCall == nil {
		return fmt.Errorf("function call is nil")
	}

	// Convert function call to tools.FunctionCall
	argsBytes, err := json.Marshal(funcCall.Args)
	if err != nil {
		return fmt.Errorf("failed to marshal function arguments: %v", err)
	}

	f, err := tools.ToFunctionCall(funcCall.Name, argsBytes)
	if err != nil {
		return fmt.Errorf("failed to create function call: %v", err)
	}

	var toolResult string
	ns := s.GetConfig()[ocstack.NAMESPACE]

	// MCP tools take priority - check MCP first
	if mcpRegistry := s.GetMCPRegistry(); mcpRegistry != nil && mcpRegistry.IsToolFromMCP(f.Name) {
		// ALWAYS override namespace parameter with ocstack's configured namespace
		if f.Arguments == nil {
			f.Arguments = make(map[string]interface{})
		}
		f.Arguments["namespace"] = ns
		// Execute MCP tool (preferred)
		toolResult = mcpRegistry.ExecuteMCPTool(f)
		f.Result = toolResult
	} else {
		// Tool not found in MCP
		toolResult = fmt.Sprintf("Tool '%s' not found in MCP", f.Name)
		f.Result = toolResult
	}

	if s.Debug {
		fmt.Printf("[DEBUG] |-->> %s\n", f.Name)
		fmt.Printf("[DEBUG] | -->> out: %s\n", f.Result)
	}

	// Create function response content
	funcResp := &genai.FunctionResponse{
		Name: funcCall.Name,
		Response: map[string]any{
			"result": toolResult,
		},
	}

	// Add function response to conversation
	respContent := &genai.Content{
		Parts: []*genai.Part{genai.NewPartFromFunctionResponse(funcCall.Name, funcResp.Response)},
		Role: "function",
	}
	contents = append(contents, respContent)

	// Continue conversation with function result
	resp, err := c.client.Models.GenerateContent(ctx, MODEL, contents, nil)
	if err != nil {
		return fmt.Errorf("failed to generate content after function call: %v", err)
	}

	// Process follow-up response
	if resp != nil && len(resp.Candidates) > 0 {
		candidate := resp.Candidates[0]
		if candidate.Content != nil && len(candidate.Content.Parts) > 0 {
			for _, part := range candidate.Content.Parts {
				if part.Text != "" {
					result := part.Text
					fmt.Printf("T :> %s\n", result)
				}
			}
		}
	}

	return nil
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
