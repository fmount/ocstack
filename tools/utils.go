package tools

import (
	"fmt"
	"github.com/fmount/ocstack/pkg/ocstack"
	"os"
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
		ocstack.ShowWarn(fmt.Sprintf("[WARN]: %v\n", err))
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

// Hello - a simple, dummy function to test the LLM ability to call functions
func Hello(args map[string]any) string {
	return fmt.Sprintf("Hello %s\n", args["name"])
}

// OC - Run openshift client tool
func OC(f *FunctionCall) string {
	args := unpackArgs("command", f.Arguments)
	res, _ := ExecTool(f.Name, args)
	return res.ToString()
}
