package handlers

import (
	"bytes"
	"context"
	"embed"
	"fmt"
	"text/template"

	"github.com/mark3labs/mcp-go/mcp"
)

//go:embed templates/*
var templatesFS embed.FS

func HandleMedicalReasoningPrompt(ctx context.Context, req mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
	arguments := req.Params.Arguments

	promptText, err := buildMedicalReasoningPrompt(arguments["user_question"], arguments["search_results"], arguments["search_status"])

	if err != nil {
		return nil, err
	}

	if promptText == "" {
		return nil, fmt.Errorf("prompt text is empty")
	}

	return &mcp.GetPromptResult{
		Description: "Medical reasoning analysis for health search results",
		Messages: []mcp.PromptMessage{
			{
				Role: "user",
				Content: mcp.TextContent{
					Type: "text",
					Text: promptText,
				},
			},
		},
	}, nil
}

func buildMedicalReasoningPrompt(userQuestion, searchResults, searchStatus string) (string, error) {
	tmpl, err := template.ParseFS(templatesFS, "templates/medical_reasoning_prompt.md")
	if err != nil {
		return "", fmt.Errorf("failed to parse medical reasoning prompt template: %w", err)
	}

	var buf bytes.Buffer
	err = tmpl.Execute(&buf, map[string]string{
		"userQuestion":  userQuestion,
		"searchStatus":  searchStatus,
		"searchResults": searchResults,
	})

	if err != nil {
		return "", fmt.Errorf("failed to execute medical reasoning prompt template: %w", err)
	}

	return buf.String(), nil
}
