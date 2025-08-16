package prompts

import (
	"bytes"
	"embed"
	"text/template"
)

//go:embed templates/*
var templatesFS embed.FS

// RenderSummarizationPrompt renders the summarization prompt using embedded Go templates
func RenderSummarizationPrompt(query, content, toolInputs string) (systemPrompt, userPrompt string, err error) {
	// Load and parse system prompt template from embedded file
	systemTemplateContent, err := templatesFS.ReadFile("templates/summarize_context_system.md")
	if err != nil {
		return "", "", err
	}

	systemTmpl, err := template.New("summarize_system").Parse(string(systemTemplateContent))
	if err != nil {
		return "", "", err
	}

	data := struct {
		Query      string
		Content    string
		ToolInputs string
	}{
		Query:      query,
		Content:    content,
		ToolInputs: toolInputs,
	}

	var systemBuf bytes.Buffer
	if err := systemTmpl.Execute(&systemBuf, data); err != nil {
		return "", "", err
	}

	// Load and parse user prompt template from embedded file
	userTemplateContent, err := templatesFS.ReadFile("templates/summarize_context_user.md")
	if err != nil {
		return "", "", err
	}

	userTmpl, err := template.New("summarize_user").Parse(string(userTemplateContent))
	if err != nil {
		return "", "", err
	}

	var userBuf bytes.Buffer
	if err := userTmpl.Execute(&userBuf, data); err != nil {
		return "", "", err
	}

	return systemBuf.String(), userBuf.String(), nil
}
