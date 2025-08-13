package prompts

import (
	"bytes"
	"embed"
	"text/template"
)

//go:embed templates/*
var templatesFS embed.FS

// ToolSelectionPromptData holds the data for tool selection prompt template
type ToolSelectionPromptData struct {
	ToolDescriptions []string
	MaxTools         int
	Query            string
	Context          string
}

// InferenceWithToolPromptData holds the data for the unified inference with tool prompt template
type InferenceWithToolPromptData struct {
	ToolDescriptions    []string
	MaxTools            int
	Query               string
	Context             string
	CurrentTurn         int
	MaxTurns            int
	PreviousToolResults string
}

// SummarizationPromptData holds the data for summarization prompt template
type SummarizationPromptData struct {
	Query   string
	Content string
}

// RenderToolSelectionPrompt renders the tool selection prompt using embedded Go templates
func RenderToolSelectionPrompt(data ToolSelectionPromptData) (systemPrompt, userPrompt string, err error) {
	// Load and parse system prompt template from embedded file
	systemTemplateContent, err := templatesFS.ReadFile("templates/tool_selection_system.md")
	if err != nil {
		return "", "", err
	}

	systemTmpl, err := template.New("system").Parse(string(systemTemplateContent))
	if err != nil {
		return "", "", err
	}

	var systemBuf bytes.Buffer
	if err := systemTmpl.Execute(&systemBuf, data); err != nil {
		return "", "", err
	}

	// Load and parse user prompt template from embedded file
	userTemplateContent, err := templatesFS.ReadFile("templates/tool_selection_user.md")
	if err != nil {
		return "", "", err
	}

	userTmpl, err := template.New("user").Parse(string(userTemplateContent))
	if err != nil {
		return "", "", err
	}

	var userBuf bytes.Buffer
	if err := userTmpl.Execute(&userBuf, data); err != nil {
		return "", "", err
	}

	return systemBuf.String(), userBuf.String(), nil
}

// RenderSummarizationPrompt renders the summarization prompt using embedded Go templates
func RenderSummarizationPrompt(data SummarizationPromptData) (systemPrompt, userPrompt string, err error) {
	// Load and parse system prompt template from embedded file
	systemTemplateContent, err := templatesFS.ReadFile("templates/summarize_context_system.md")
	if err != nil {
		return "", "", err
	}

	systemTmpl, err := template.New("summarize_system").Parse(string(systemTemplateContent))
	if err != nil {
		return "", "", err
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

// RenderInferenceWithToolPrompt renders the unified inference with tool prompt using embedded Go templates
func RenderInferenceWithToolPrompt(data InferenceWithToolPromptData) (systemPrompt, userPrompt string, err error) {
	// Load and parse system prompt template from embedded file
	systemTemplateContent, err := templatesFS.ReadFile("templates/inference_with_tool_system.md")
	if err != nil {
		return "", "", err
	}

	systemTmpl, err := template.New("inference_system").Parse(string(systemTemplateContent))
	if err != nil {
		return "", "", err
	}

	var systemBuf bytes.Buffer
	if err := systemTmpl.Execute(&systemBuf, data); err != nil {
		return "", "", err
	}

	// Load and parse user prompt template from embedded file
	userTemplateContent, err := templatesFS.ReadFile("templates/inference_with_tool_user.md")
	if err != nil {
		return "", "", err
	}

	userTmpl, err := template.New("inference_user").Parse(string(userTemplateContent))
	if err != nil {
		return "", "", err
	}

	var userBuf bytes.Buffer
	if err := userTmpl.Execute(&userBuf, data); err != nil {
		return "", "", err
	}

	return systemBuf.String(), userBuf.String(), nil
}
