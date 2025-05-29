package tools

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"text/template"
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

// GetTools -
func RegisterTools() ([]byte, error) {
	// Define the hello tool
	helloTool := map[string]any{
		"type": "function",
		"function": map[string]any{
			"name":        "hello",
			"description": "Say hello to a given person with his name",
			"parameters": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"name": map[string]any{
						"type":        "string",
						"description": "The name of the person",
					},
				},
				"required": []string{"name"},
			},
		},
	}
	ocTool := map[string]any{
		"type": "function",
		"function": map[string]any{
			"name":        "oc",
			"description": "Runs the openshift client (oc) to interact with an openshift environment",
		},
	}
	ctlplaneTool := map[string]any{
		"type": "function",
		"function": map[string]any{
			"name":        "get_openstack_control_plane",
			"description": "Runs the openshift client (oc) to get the openstack control plane status",
		},
	}
	jt, err := json.Marshal([]any{ctlplaneTool, ocTool, helloTool})
	if err != nil {
		return nil, err
	}
	return jt, nil
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
