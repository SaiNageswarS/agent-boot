package prompts

import (
	"bytes"
	"embed"
	"html/template"
)

//go:embed templates/*
var templatesFS embed.FS

func loadPrompt(templatePath string, data interface{}) (string, error) {
	tmpl, err := template.ParseFS(templatesFS, templatePath)
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	err = tmpl.Execute(&buf, data)
	if err != nil {
		return "", err
	}

	return buf.String(), nil
}
