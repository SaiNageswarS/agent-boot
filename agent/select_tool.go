package agent

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/SaiNageswarS/agent-boot/llm"
	"github.com/SaiNageswarS/agent-boot/prompts"
	"github.com/SaiNageswarS/agent-boot/schema"
	"github.com/SaiNageswarS/go-api-boot/logger"
	"go.uber.org/zap"
)

// ToolSelectionRequest represents a request for tool selection
type ToolSelectionRequest struct {
	Query       string            `json:"query"`
	Context     string            `json:"context,omitempty"`
	MaxTools    int               `json:"max_tools,omitempty"`
	Preferences map[string]string `json:"preferences,omitempty"`
}

// SelectTools uses the mini model to intelligently select appropriate tools for a given query
func (a *Agent) SelectTools(ctx context.Context, req ToolSelectionRequest) ([]*schema.SelectedTool, error) {
	if len(a.config.Tools) == 0 {
		return []*schema.SelectedTool{}, nil
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
	err = a.config.MiniModel.GenerateInference(
		ctx,
		messages,
		func(chunk string) error {
			responseContent.WriteString(chunk)
			return nil
		},
		llm.WithTemperature(0.3),
		llm.WithMaxTokens(1000),
	)

	if err != nil {
		return nil, fmt.Errorf("failed to select tools: %w", err)
	}

	// Parse the response to extract tool selections
	selections, err := a.parseToolSelections(responseContent.String())
	if err != nil {
		logger.Error("Failed to parse tool selections", zap.Error(err), zap.String("response", responseContent.String()))
		// Fallback: return the first tool as a basic selection
		if len(a.config.Tools) > 0 {
			return []*schema.SelectedTool{
				{
					Name:       a.config.Tools[0].Name,
					Parameters: make(map[string]string),
					Query:      req.Query,
				},
			}, nil
		}
	}

	for _, selection := range selections {
		// Set the query for each selected tool
		selection.Query = req.Query
	}

	return selections, nil
}

func (a *Agent) parseToolSelections(response string) ([]*schema.SelectedTool, error) {
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
	selections := make([]*schema.SelectedTool, 0, len(toolBlocks))

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
func (a *Agent) parseToolBlock(block string) (*schema.SelectedTool, error) {
	lines := strings.Split(block, "\n")

	var toolName string
	var confidence float64 = 0.5
	var reasoning string
	parameters := make(map[string]string)

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
	logger.Info("Selected tool",
		zap.String("tool_name", toolName),
		zap.String("reason", reasoning),
		zap.Float64("confidence", confidence))

	return &schema.SelectedTool{
		Name:       toolName,
		Parameters: parameters,
	}, nil
}
