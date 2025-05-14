package templates

import (
	"bytes"
	"embed"
	"fmt"
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
