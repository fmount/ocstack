package tools

import (
	"fmt"
	"github.com/fmount/ocstack/pkg/ocstack"
	"os"
)

// LoadDefaultConfig -
func LoadDefaultConfig() map[string]string {
	c := make(map[string]string)
	c[ocstack.NAMESPACE] = ocstack.DEFAULT_NAMESPACE
	return c
}

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

// OC - Run openshift client tool
func OC(f *FunctionCall) string {
	args := unpackArgs("command", f.Arguments)
	res, _ := ExecTool(f.Name, args)
	return res.ToString()
}

// Ctlplane -
func Ctlplane(f *FunctionCall, ns string) string {
	//args := unpackArgs("command", f.Arguments)
	res, _ := ExecTool("oc", fmt.Sprintf("-n %s get oscp", ns))
	return res.ToString()
}

// Check service -
func CheckSvc(f *FunctionCall, ns string) string {
	svc := unpackArgs("service", f.Arguments)
	res, _ := ExecTool("oc", fmt.Sprintf("-n %s get %s", ns, svc))
	return res.ToString()
}

func GetDeployedVersion(f *FunctionCall, ns string) string {
	options := "-o custom-columns=VERSION:.status.deployedVersion --no-headers"
	res, _ := ExecTool("oc", fmt.Sprintf("-n %s get openstackversion %s", ns, options))
	return res.ToString()
}

func GetAvailableVersion(f *FunctionCall, ns string) string {
	options := "-o custom-columns=.VERSION:.status.availableVersion --no-headers"
	res, _ := ExecTool("oc", fmt.Sprintf("-n %s get openstackversion %s", ns, options))
	return res.ToString()
}

func MinorUpdate(f *FunctionCall, ns string) string {
	av := GetAvailableVersion(f, ns)
	dv := GetDeployedVersion(f, ns)
	if av == dv {
		return "OpenStack is up to date"
	}
	return "OpenStack control update available!"
}

func TriggerUpdate(f *FunctionCall, ns string, name string, targetVersion string) string {
	res, _ := ExecTool("oc", fmt.Sprintf("-n %s patch openstackversion %s --type=json -p=\"[{'op': 'replace', 'path': '/spec/targetVersion', '%s'}]\"", ns, name, targetVersion))
	return res.ToString()
}
