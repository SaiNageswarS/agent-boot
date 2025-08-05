package agent

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/SaiNageswarS/agent-boot/llm"
	"github.com/SaiNageswarS/agent-boot/schema"
	"github.com/SaiNageswarS/go-api-boot/logger"
	"go.uber.org/zap"
)

// PromptTemplate represents a reusable prompt template
type PromptTemplate struct {
	Name      string            `json:"name"`
	Template  string            `json:"template"`
	Variables []string          `json:"variables"`
	Metadata  map[string]string `json:"metadata"`
}

// AgentConfig holds configuration for the agent
type AgentConfig struct {
	MiniModel struct {
		Client llm.LLMClient
		Model  string
	}
	BigModel struct {
		Client llm.LLMClient
		Model  string
	}
	Tools     []MCPTool
	Prompt    PromptTemplate
	MaxTokens int
}

// Agent represents the main agent system
type Agent struct {
	schema.UnimplementedAgentServer
	config AgentConfig
}

// NewAgent creates a new agent instance
func NewAgent(config AgentConfig) *Agent {
	return &Agent{
		config: config,
	}
}

// GenerateAnswer is the main API to generate answers using tool selection and prompts
func (a *Agent) Execute(ctx context.Context, reporter ProgressReporter, req *schema.GenerateAnswerRequest) (*schema.AnswerChunk, error) {
	startTime := getCurrentTimeMs()

	response := &schema.AnswerChunk{
		ToolsUsed: make([]string, 0),
		Metadata:  make(map[string]string),
	}

	var toolResults []string

	// Step 1: Select and execute tools if requested
	if len(a.config.Tools) > 0 {
		toolSelectionReq := ToolSelectionRequest{
			Query:    req.Question,
			Context:  req.Context,
			MaxTools: 3, // Default max tools
		}

		reporter.Send(NewProgressUpdate(
			schema.Stage_tool_selection_starting,
			"Selecting tools for query: "+req.Question,
			1,
		))

		selectedTools, err := a.SelectTools(ctx, toolSelectionReq)
		if err != nil {
			// Tool selection failed, continuing without tools
			logger.Error("Tool selection failed", zap.Error(err))
			reporter.Send(NewStreamError(err.Error(), "Tool selection error"))
		} else {
			reporter.Send(NewProgressUpdate(
				schema.Stage_tool_selection_completed,
				fmt.Sprintf("Selected %d tools for query: %s", len(selectedTools), req.Question),
				1,
			))

			for _, selection := range selectedTools {
				reporter.Send(NewToolSelectionResult(selection))
			}

			// Execute selected tools
			for _, selection := range selectedTools {
				reporter.Send(NewProgressUpdate(
					schema.Stage_tool_execution_starting,
					fmt.Sprintf("Executing tool: %s", selection.Name),
					2,
				))

				toolResultChan := a.ExecuteTool(ctx, selection)

				// Process each result from the tool
				for toolResult := range toolResultChan {
					if toolResult.Error != "" {
						logger.Error("Tool execution error", zap.String("tool", selection.Name), zap.Error(errors.New(toolResult.Error)))
						reporter.Send(NewStreamError(toolResult.Error, fmt.Sprintf("Error executing tool %s", selection.Name)))
						continue
					}

					reporter.Send(NewToolExecutionResult(selection.Name, toolResult))

					if len(toolResult.Sentences) > 0 {
						// Format the tool result for inclusion in the prompt
						toolResultText := a.formatToolResult(selection.Name, toolResult)
						toolResults = append(toolResults, toolResultText)
						response.ToolsUsed = append(response.ToolsUsed, selection.Name)
					}
				}
			}
		}
	}

	// Step 2: Get or create the prompt
	reporter.Send(NewProgressUpdate(
		schema.Stage_answer_generation_starting,
		"Generating answer...",
		3,
	))

	prompt := a.getPrompt(req.Question, req.Context, toolResults)
	response.PromptUsed = prompt

	// Step 3: Decide which model to use based on complexity
	useBigModel := a.shouldUseBigModel(req.Question, toolResults)
	var client llm.LLMClient
	var modelName string

	if useBigModel {
		client = a.config.BigModel.Client
		modelName = a.config.BigModel.Model
	} else {
		client = a.config.MiniModel.Client
		modelName = a.config.MiniModel.Model
	}

	response.ModelUsed = modelName

	// Step 4: Generate the final answer
	finalAnswer, err := a.GenerateAnswer(ctx, client, modelName, prompt)
	if err != nil {
		logger.Error("Answer generation failed", zap.Error(err))
		reporter.Send(NewStreamError(err.Error(), "Answer generation error"))
		return nil, err
	}

	response.Answer = finalAnswer
	response.ProcessingTime = getCurrentTimeMs() - startTime

	// Add metadata
	response.Metadata["tool_count"] = strconv.Itoa(len(response.ToolsUsed))
	response.Metadata["has_context"] = strconv.FormatBool(req.Context != "")
	response.Metadata["used_big_model"] = strconv.FormatBool(useBigModel)
	response.IsFinal = true

	reporter.Send(NewProgressUpdate(
		schema.Stage_answer_generation_completed,
		"Answer generation completed successfully",
		3,
	))

	reporter.Send(NewAnswerChunk(response))
	reporter.Send(NewStreamComplete("Answer generation completed"))
	return response, nil
}

func (a *Agent) GetToolByName(name string) *MCPTool {
	for _, tool := range a.config.Tools {
		if tool.Name == name {
			return &tool
		}
	}
	return nil
}

func (a *Agent) getPrompt(query, context string, toolResults []string) string {
	if a.config.Prompt.Template != "" {
		return a.fillPromptTemplate(a.config.Prompt, map[string]string{
			"query":        query,
			"context":      context,
			"tool_results": strings.Join(toolResults, "\n"),
		})
	}

	// Default prompt
	prompt := fmt.Sprintf("Query: %s", query)

	if context != "" {
		prompt += fmt.Sprintf("\n\nContext: %s", context)
	}

	if len(toolResults) > 0 {
		prompt += fmt.Sprintf("\n\nTool Results:\n%s", strings.Join(toolResults, "\n"))
	}

	prompt += "\n\nPlease provide a comprehensive answer based on the above information."

	return prompt
}

func (a *Agent) fillPromptTemplate(template PromptTemplate, variables map[string]string) string {
	result := template.Template

	for key, value := range variables {
		placeholder := fmt.Sprintf("{{%s}}", key)
		result = strings.ReplaceAll(result, placeholder, value)
	}

	return result
}

func (a *Agent) shouldUseBigModel(query string, toolResults []string) bool {
	// Simple heuristics to decide when to use the big model

	// Use big model for complex queries (longer than 100 chars)
	if len(query) > 100 {
		return true
	}

	// Use big model when we have tool results to synthesize
	if len(toolResults) > 1 {
		return true
	}

	// Use big model for certain keywords indicating complexity
	complexKeywords := []string{
		"analyze", "compare", "summarize", "explain", "detailed",
		"comprehensive", "research", "investigate", "complex",
	}

	queryLower := strings.ToLower(query)
	for _, keyword := range complexKeywords {
		if strings.Contains(queryLower, keyword) {
			return true
		}
	}

	return false
}

// GetAvailableTools returns a list of all available tools
func (a *Agent) GetAvailableTools() []MCPTool {
	return a.config.Tools
}

// formatToolResult formats a ToolResult into a human-readable string for prompts
func (a *Agent) formatToolResult(toolName string, result *schema.ToolExecutionResultChunk) string {
	var formatted strings.Builder

	formatted.WriteString(fmt.Sprintf("Tool %s result:", toolName))

	if result.Title != "" {
		formatted.WriteString(fmt.Sprintf("\nTitle: %s", result.Title))
	}

	if len(result.Sentences) > 0 {
		formatted.WriteString("\nContent:")
		for _, sentence := range result.Sentences {
			formatted.WriteString(fmt.Sprintf("\n- %s", sentence))
		}
	}

	if result.Attribution != "" {
		formatted.WriteString(fmt.Sprintf("\nSource: %s", result.Attribution))
	}

	if len(result.Metadata) > 0 {
		formatted.WriteString("\nAdditional Info:")
		for key, value := range result.Metadata {
			formatted.WriteString(fmt.Sprintf("\n- %s: %v", key, value))
		}
	}

	return formatted.String()
}

func (a *Agent) getMaxTokens() int {
	if a.config.MaxTokens > 0 {
		return a.config.MaxTokens
	}
	return 2000 // Default max tokens
}

func getMaxTools(requested int) int {
	if requested > 0 && requested <= 10 {
		return requested
	}
	return 3 // Default max tools
}

func getCurrentTimeMs() int64 {
	return time.Now().UnixMilli()
}
