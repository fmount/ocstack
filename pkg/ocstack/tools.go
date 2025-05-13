package ocstack

import (
	"fmt"
	"os"

	"github.com/ollama/ollama/api"
)

type Tools struct {
	Tools []api.Tool
}

// TODO Read KUBECONFIG to setup a k8s client
// k8s will be used to rsh to the openstackclient Pod that is used to
// interact with openstack
// Note: this information has to be part of the template that we load
// as context for the LLM

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
