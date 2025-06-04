package tools

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/template"
)

const (
	LOCAL_TOOLS = "tools/local"
)

type Tool struct {
	Type     string    `json:"type"`
	Function *Function `json:"function"`
}

type FunctionCall struct {
	Name      string         `json:"name"`
	Arguments map[string]any `json:"arguments"`
	Result    string         `json:"result"`
}

type Properties struct {
	Type        string   `json:"type,omitempty"`
	Description string   `json:"description,omitempty"`
	Enum        []string `json:"enum,omitempty"`
}

type Parameters struct {
	Type       string                 `json:"type,omitempty"`
	Required   []string               `json:"required,omitempty"`
	Properties map[string]*Properties `json:"properties,omitempty"`
}

type Function struct {
	Name        string      `json:"name,omitempty"`
	Description string      `json:"description,omitempty"`
	Parameters  *Parameters `json:"parameters,omitempty"`
}

// ToFunctionArgs -
func ToFunctionArgs(b []byte) (map[string]any, error) {
	m := make(map[string]any)
	err := json.Unmarshal(b, &m)
	if err != nil {
		return nil, fmt.Errorf("Can't unmarshal data")
	}
	return m, nil
}

func ToFunctionCall(name string, b []byte) (*FunctionCall, error) {
	var err error
	var args map[string]any
	if args, err = ToFunctionArgs(b); err != nil {
		return nil, err
	}
	f := &FunctionCall{
		Name:      name,
		Arguments: args,
	}
	return f, nil
}

func RenderExec(f *FunctionCall) (string, error) {
	tmpl, err := template.ParseFiles("template/resources/execResult.tmpl")
	if err != nil {
		return "", fmt.Errorf("Error parsing template file: %v", err)
	}
	var buf bytes.Buffer
	// Execute the template with the data and write the output to stdout
	err = tmpl.Execute(&buf, f)
	if err != nil {
		return "", fmt.Errorf("Error executing template: %v", err)
	}
	return buf.String(), nil
}

func unpackArgs(key string, args map[string]any) string {
	// only return the value if the key exists and is a .(string)
	if arg, exists := args[key]; exists {
		if argStr, ok := arg.(string); ok {
			return argStr
		}
		return ""
	}
	return ""
}

// RegisterTools - A function that either select local tools or simply
// discover what is available through and endpoint. Currently local tools
// only are supported
func GetRegisteredTools(dirPath string) ([]byte, error) {

	var allTools []map[string]any
	// Read all JSON files from the directory
	err := filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories and non-JSON files
		if info.IsDir() || !strings.HasSuffix(strings.ToLower(info.Name()), ".json") {
			return nil
		}

		// Read the JSON file
		data, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("failed to read file %s: %w", path, err)
		}

		// Parse the JSON
		var tools []map[string]any
		if err := json.Unmarshal(data, &tools); err != nil {
			return fmt.Errorf("failed to parse JSON from %s: %w", path, err)
		}

		// Merge into the main slice
		allTools = append(allTools, tools...)
		return nil
	})

	if err != nil {
		return nil, err
	}

	// Marshal the merged tools back to JSON
	return json.Marshal(allTools)
}

// RegisterTools - A function that either select local tools or simply
// discover what is available through and endpoint. Currently local tools
// only are supported
func RegisterTools() ([]byte, error) {
	// Read the JSON files from local dir
	return GetRegisteredTools(LOCAL_TOOLS)
}

// ToolResult -
type ToolResult struct {
	Stdout   string `json:"stdout,omitempty"`
	Stderr   string `json:"stderr,omitempty"`
	ExitCode int    `json:"exitcode,omitempty"`
}

// ToString -
func (t *ToolResult) ToString() string {
	return fmt.Sprintf("out: %s\nerr: %s\n", t.Stdout, t.Stderr)
}

// ExecTool executes a command with arguments and returns the result
func ExecTool(c string, args string) (ToolResult, error) {
	// Split the args string into separate arguments
	argSlice := strings.Fields(args)

	cmd := exec.Command(c, argSlice...)

	// Define stdout and stderr buffers to collect the execution result
	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	t := ToolResult{}
	err := cmd.Run()
	if err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			t.ExitCode = exitError.ExitCode()
		}
		t.Stdout = stdout.String()
		t.Stderr = stderr.String()
		return t, fmt.Errorf("command failed with error: %s", err)
	}

	t.Stdout = stdout.String()
	t.Stderr = stderr.String()
	fmt.Println(stdout.String())
	t.ExitCode = 0
	return t, nil
}
