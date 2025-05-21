package templates

import (
	"bytes"
	"embed"
	"fmt"
	"github.com/fmount/ocstack/llm"
	"text/template"
)

//go:embed resources/*.tmpl
var templateFS embed.FS

// Template parameters used to customize a given template
type AgentParams struct {
	UseTools bool
}

// LoadProfile -
func LoadProfile(templateName string) (string, error) {
	// Parse all templates under template/resources at once
	tmpl, err := template.ParseFS(templateFS, "resources/*.tmpl")
	if err != nil {
		return "", fmt.Errorf("Malformed template: %w", err)
	}
	a := AgentParams{true}
	var tpl bytes.Buffer
	tmplExt := fmt.Sprintf("%s.tmpl", templateName)
	if err := tmpl.ExecuteTemplate(
		&tpl,
		tmplExt,
		a,
	); err != nil {
		return "", err
	}
	return tpl.String(), nil
}

// RenderExec -
func RenderExec(f llm.FunctionCall) (string, error) {
	tmpl, err := template.ParseFiles("resources/execResult.tmpl")
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
