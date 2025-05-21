package ocstack

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// GetKubeConfig -
func GetKubeConfig() (string, error) {
	path := os.Getenv("KUBECONFIG")
	if path == "" {
		return "", fmt.Errorf("KUBECONFIG env var is not set")
	}
	return path, nil
}

// ValidateTools -
func ValidateTools() []error {
	var allErrs []error
	// Try to resolve KubeConfig
	if _, err := GetKubeConfig(); err != nil {
		ShowWarn(fmt.Sprintf("[WARN]: %v\n", err))
		allErrs = append(allErrs, err)
	}
	return allErrs
}

// ExitOnErrors -
func ExitOnErrors() {
	errors := ValidateTools()
	if len(errors) > 0 {
		os.Exit(1)
	}
}

// GetTools -
func RegisterTools() ([]byte, error) {
	// Define the hello tool
	/*helloTool := map[string]any{
		"type": "function",
		"function": map[string]any{
			"name":		   "hello",
			"description": "Say hello to a given person with his name",
			"parameters": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"name": map[string]any{
						"type":		   "string",
						"description": "The name of the person",
					},
				},
				"required": []string{"name"},
			},
		},
	}*/
	ocTool := map[string]any{
		"type": "function",
		"function": map[string]any{
			"name":        "oc",
			"description": "Runs the openshift client (oc) to interact with an openshift environment",
			/*"parameters": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"namespace": map[string]any{
						"type":		   "string",
						"description": "The namespace where you want to execute your query",
					},
				},
				"required": []string{"name"},
			},*/
		},
	}
	jt, err := json.Marshal([]any{ocTool})
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
