package agent

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/SaiNageswarS/agent-boot/core/llm"
	"github.com/SaiNageswarS/agent-boot/core/prompts"
)

// ToolSelectionRequest represents a request for tool selection
type ToolSelectionRequest struct {
	Query       string            `json:"query"`
	Context     string            `json:"context,omitempty"`
	MaxTools    int               `json:"max_tools,omitempty"`
	Preferences map[string]string `json:"preferences,omitempty"`
}

// ToolSelection represents a selected tool with parameters
type ToolSelection struct {
	Tool       MCPTool                `json:"tool"`
	Parameters map[string]interface{} `json:"parameters"`
	Confidence float64                `json:"confidence"`
	Reasoning  string                 `json:"reasoning"`
}

// SelectTools uses the mini model to intelligently select appropriate tools for a given query
func (a *Agent) SelectTools(ctx context.Context, req ToolSelectionRequest) ([]ToolSelection, error) {
	if len(a.config.Tools) == 0 {
		return []ToolSelection{}, nil
	}

	// Create tool descriptions for the prompt
	toolDescriptions := make([]string, 0, len(a.config.Tools))
	for _, tool := range a.config.Tools {
		toolDescriptions = append(toolDescriptions, fmt.Sprintf("- %s: %s", tool.Name, tool.Description))
	}

	// Prepare data for template rendering
	promptData := prompts.ToolSelectionPromptData{
		ToolDescriptions: toolDescriptions,
		MaxTools:         getMaxTools(req.MaxTools),
		Query:            req.Query,
		Context:          req.Context,
	}

	// Render the prompts using embedded Go templates
	systemPrompt, userPrompt, err := prompts.RenderToolSelectionPrompt(promptData)
	if err != nil {
		return nil, fmt.Errorf("failed to render tool selection prompt: %w", err)
	}

	messages := []llm.Message{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: userPrompt},
	}

	var responseContent strings.Builder
	err = a.config.MiniModel.Client.GenerateInference(
		ctx,
		messages,
		func(chunk string) error {
			responseContent.WriteString(chunk)
			return nil
		},
		llm.WithLLMModel(a.config.MiniModel.Model),
		llm.WithTemperature(0.3),
		llm.WithMaxTokens(1000),
	)

	if err != nil {
		return nil, fmt.Errorf("failed to select tools: %w", err)
	}

	// Parse the response to extract tool selections
	selections, err := a.parseToolSelections(responseContent.String())
	if err != nil {
		// Fallback: return the first tool as a basic selection
		if len(a.config.Tools) > 0 {
			return []ToolSelection{
				{
					Tool:       a.config.Tools[0],
					Parameters: make(map[string]interface{}),
					Confidence: 0.5,
					Reasoning:  "Fallback selection due to parsing error",
				},
			}, nil
		}
	}

	return selections, nil
}

func (a *Agent) parseToolSelections(response string) ([]ToolSelection, error) {
	// Look for TOOL_SELECTION_START and TOOL_SELECTION_END markers
	startMarker := "TOOL_SELECTION_START"
	endMarker := "TOOL_SELECTION_END"

	startIdx := strings.Index(response, startMarker)
	endIdx := strings.Index(response, endMarker)

	if startIdx == -1 || endIdx == -1 || startIdx >= endIdx {
		return nil, fmt.Errorf("no valid tool selection block found")
	}

	// Extract the content between markers
	content := response[startIdx+len(startMarker) : endIdx]

	// Split into tool blocks
	toolBlocks := strings.Split(content, "TOOL:")
	selections := make([]ToolSelection, 0, len(toolBlocks))

	for _, block := range toolBlocks {
		block = strings.TrimSpace(block)
		if block == "" {
			continue
		}

		selection, err := a.parseToolBlock(block)
		if err != nil {
			continue // Skip invalid blocks
		}

		selections = append(selections, selection)
	}

	return selections, nil
}

// parseToolBlock parses an individual tool block
func (a *Agent) parseToolBlock(block string) (ToolSelection, error) {
	lines := strings.Split(block, "\n")

	var toolName string
	var confidence float64 = 0.5
	var reasoning string
	parameters := make(map[string]interface{})

	inParameters := false

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		if !inParameters {
			// Parse tool name (first line should be the tool name)
			if toolName == "" {
				toolName = strings.TrimSpace(line)
				continue
			}

			// Parse confidence
			if strings.HasPrefix(line, "CONFIDENCE:") {
				confidenceStr := strings.TrimSpace(strings.TrimPrefix(line, "CONFIDENCE:"))
				if c, err := strconv.ParseFloat(confidenceStr, 64); err == nil {
					confidence = c
				}
				continue
			}

			// Parse reasoning
			if strings.HasPrefix(line, "REASONING:") {
				reasoning = strings.TrimSpace(strings.TrimPrefix(line, "REASONING:"))
				continue
			}

			// Check for parameters section
			if strings.HasPrefix(line, "PARAMETERS:") {
				inParameters = true
				continue
			}
		} else {
			// Parse parameter lines
			if strings.Contains(line, ":") {
				parts := strings.SplitN(line, ":", 2)
				if len(parts) == 2 {
					key := strings.TrimSpace(parts[0])
					value := strings.TrimSpace(parts[1])
					parameters[key] = value
				}
			}
		}
	}

	// Find the tool by name
	var foundTool *MCPTool
	for _, tool := range a.config.Tools {
		if tool.Name == toolName {
			foundTool = &tool
			break
		}
	}

	if foundTool == nil {
		return ToolSelection{}, fmt.Errorf("tool not found: %s", toolName)
	}

	if reasoning == "" {
		reasoning = "Selected by AI"
	}

	return ToolSelection{
		Tool:       *foundTool,
		Parameters: parameters,
		Confidence: confidence,
		Reasoning:  reasoning,
	}, nil
}
