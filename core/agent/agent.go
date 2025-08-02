package agent

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/SaiNageswarS/agent-boot/core/llm"
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
	Prompts   map[string]PromptTemplate
	MaxTokens int
}

// Agent represents the main agent system
type Agent struct {
	config AgentConfig
}

// NewAgent creates a new agent instance
func NewAgent(config AgentConfig) *Agent {
	return &Agent{
		config: config,
	}
}

// GenerateAnswerRequest represents a request for answer generation
type GenerateAnswerRequest struct {
	Query          string            `json:"query"`
	Context        string            `json:"context,omitempty"`
	PromptTemplate string            `json:"prompt_template,omitempty"`
	UseTools       bool              `json:"use_tools"`
	MaxIterations  int               `json:"max_iterations,omitempty"`
	Metadata       map[string]string `json:"metadata,omitempty"`
}

// GenerateAnswerResponse represents the response from answer generation
type GenerateAnswerResponse struct {
	Answer         string                 `json:"answer"`
	ToolsUsed      []ToolSelection        `json:"tools_used"`
	PromptUsed     string                 `json:"prompt_used"`
	ModelUsed      string                 `json:"model_used"`
	TokensUsed     int                    `json:"tokens_used"`
	ProcessingTime int64                  `json:"processing_time_ms"`
	Metadata       map[string]interface{} `json:"metadata,omitempty"`
}

// GenerateAnswer is the main API to generate answers using tool selection and prompts
func (a *Agent) GenerateAnswer(ctx context.Context, req GenerateAnswerRequest) (*GenerateAnswerResponse, error) {
	startTime := getCurrentTimeMs()

	response := &GenerateAnswerResponse{
		ToolsUsed: make([]ToolSelection, 0),
		Metadata:  make(map[string]interface{}),
	}

	var toolResults []string

	// Step 1: Select and execute tools if requested
	if req.UseTools {
		toolSelectionReq := ToolSelectionRequest{
			Query:    req.Query,
			Context:  req.Context,
			MaxTools: 3, // Default max tools
		}

		selectedTools, err := a.SelectTools(ctx, toolSelectionReq)
		if err != nil {
			// Tool selection failed, continuing without tools
		} else {
			// Execute selected tools
			for _, selection := range selectedTools {
				// Add user query to parameters for summarization context
				if selection.Tool.SummarizeContext {
					if selection.Parameters == nil {
						selection.Parameters = make(map[string]interface{})
					}
					// Only add query if not already present
					if _, exists := selection.Parameters["query"]; !exists {
						selection.Parameters["query"] = req.Query
					}
				}

				results, err := a.ExecuteTool(ctx, selection)
				if err != nil {
					// Tool execution failed, skip this tool
					continue
				}

				// Process each result from the tool
				for _, result := range results {
					toolResultText := a.formatToolResult(selection.Tool.Name, result)
					toolResults = append(toolResults, toolResultText)
				}

				response.ToolsUsed = append(response.ToolsUsed, selection)
			}
		}
	}

	// Step 2: Get or create the prompt
	prompt := a.getPrompt(req.PromptTemplate, req.Query, req.Context, toolResults)
	response.PromptUsed = prompt

	// Step 3: Decide which model to use based on complexity
	useBigModel := a.shouldUseBigModel(req.Query, toolResults)
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
	messages := []llm.Message{
		{Role: "user", Content: prompt},
	}

	var responseContent strings.Builder
	err := client.GenerateInference(
		ctx,
		messages,
		func(chunk string) error {
			responseContent.WriteString(chunk)
			return nil
		},
		llm.WithLLMModel(modelName),
		llm.WithTemperature(0.7),
		llm.WithMaxTokens(a.getMaxTokens()),
	)

	if err != nil {
		return nil, fmt.Errorf("failed to generate answer: %w", err)
	}

	response.Answer = responseContent.String()
	response.ProcessingTime = getCurrentTimeMs() - startTime

	// Add metadata
	response.Metadata["tool_count"] = len(response.ToolsUsed)
	response.Metadata["has_context"] = req.Context != ""
	response.Metadata["used_big_model"] = useBigModel

	return response, nil
}

// AddTool adds a new MCP tool to the agent
func (a *Agent) AddTool(tool MCPTool) {
	a.config.Tools = append(a.config.Tools, tool)
}

// AddPrompt adds a new prompt template to the agent
func (a *Agent) AddPrompt(name string, template PromptTemplate) {
	if a.config.Prompts == nil {
		a.config.Prompts = make(map[string]PromptTemplate)
	}
	a.config.Prompts[name] = template
}

// GetAvailableTools returns a list of all available tools
func (a *Agent) GetAvailableTools() []MCPTool {
	return a.config.Tools
}

// GetAvailablePrompts returns a list of all available prompt templates
func (a *Agent) GetAvailablePrompts() map[string]PromptTemplate {
	return a.config.Prompts
}

// formatToolResult formats a ToolResult into a human-readable string for prompts
func (a *Agent) formatToolResult(toolName string, result *ToolResultChunk) string {
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

	if len(result.Attribution) > 0 {
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

func (a *Agent) getPrompt(templateName, query, context string, toolResults []string) string {
	if templateName != "" {
		if template, exists := a.config.Prompts[templateName]; exists {
			return a.fillPromptTemplate(template, map[string]string{
				"query":        query,
				"context":      context,
				"tool_results": strings.Join(toolResults, "\n"),
			})
		}
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
