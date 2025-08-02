package agent

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/SaiNageswarS/agent-boot/core/llm"
)

// ToolResult represents a standardized format for tool execution results
type ToolResult struct {
	// Primary content - can be multiple sentences or a single result
	Sentences []string `json:"sentences"`

	// Source attribution - where the information came from
	Attributions []string `json:"attributions,omitempty"`

	// Title or summary of the result
	Title string `json:"title,omitempty"`

	// Metadata for additional context
	Metadata map[string]interface{} `json:"metadata,omitempty"`

	// Success indicator
	Success bool `json:"success"`

	// Error message if not successful
	Error string `json:"error,omitempty"`
}

// MCPTool represents a Model Context Protocol tool
type MCPTool struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Parameters  map[string]interface{} `json:"parameters"`
	// SummarizeContext enables automatic summarization of tool results using the mini model.
	// When enabled, each ToolResult's Sentences will be summarized with respect to the user's query.
	// Irrelevant content will be filtered out, making this ideal for RAG search and web search tools.
	SummarizeContext bool `json:"summarize_context"`
	Handler          func(ctx context.Context, params map[string]interface{}) ([]*ToolResult, error)
}

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
	promptData := ToolSelectionPromptData{
		ToolDescriptions: toolDescriptions,
		MaxTools:         getMaxTools(req.MaxTools),
		Query:            req.Query,
		Context:          req.Context,
	}

	// Render the prompts using embedded Go templates
	systemPrompt, userPrompt, err := RenderToolSelectionPrompt(promptData)
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

				results, err := a.executeTool(ctx, selection)
				if err != nil {
					// Tool execution failed, skip this tool
					continue
				}

				// Process each result from the tool
				hasSuccessfulResult := false
				for _, result := range results {
					if result.Success {
						hasSuccessfulResult = true
						// Format the tool result for inclusion in the prompt
						toolResultText := a.formatToolResult(selection.Tool.Name, result)
						toolResults = append(toolResults, toolResultText)
					}
				}

				// Only add to tools used if at least one result was successful
				if hasSuccessfulResult {
					response.ToolsUsed = append(response.ToolsUsed, selection)
				}
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

// Helper methods
// parseStructuredTextSelections parses the new TOOL_SELECTION_START/END format
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

func (a *Agent) executeTool(ctx context.Context, selection ToolSelection) ([]*ToolResult, error) {
	results, err := selection.Tool.Handler(ctx, selection.Parameters)
	if err != nil {
		return nil, err
	}

	// Apply summarization if enabled for this tool
	if selection.Tool.SummarizeContext {
		// Get the user query from context metadata if available
		query := ""
		if queryParam, ok := selection.Parameters["query"].(string); ok {
			query = queryParam
		}

		summarizedResults, err := a.summarizeToolResults(ctx, results, query)
		if err != nil {
			// If summarization fails, return original results
			return results, nil
		}
		return summarizedResults, nil
	}

	return results, nil
}

// summarizeToolResults summarizes tool results using the mini model to make them more relevant and concise.
// This method:
// 1. Combines all sentences from each ToolResult into a single text
// 2. Uses the mini model to summarize the content with respect to the user's query
// 3. Filters out irrelevant content completely
// 4. Preserves original metadata and attributions
// 5. Adds summarization metadata for transparency
//
// This is particularly useful for:
// - RAG search results that may contain verbose or tangential information
// - Web search results with mixed relevant/irrelevant content
// - Large document chunks that need to be condensed for context windows
func (a *Agent) summarizeToolResults(ctx context.Context, results []*ToolResult, userQuery string) ([]*ToolResult, error) {
	if len(results) == 0 {
		return results, nil
	}

	summarizedResults := make([]*ToolResult, 0, len(results))

	for _, result := range results {
		if !result.Success || len(result.Sentences) == 0 {
			// Keep unsuccessful results or results without sentences unchanged
			summarizedResults = append(summarizedResults, result)
			continue
		}

		// Join all sentences into a single text
		combinedText := strings.Join(result.Sentences, " ")

		// Create summarization prompt using templates
		promptData := SummarizationPromptData{
			Query:   userQuery,
			Content: combinedText,
		}

		systemPrompt, userPrompt, err := RenderSummarizationPrompt(promptData)
		if err != nil {
			// If template rendering fails, keep the original result
			summarizedResults = append(summarizedResults, result)
			continue
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
			llm.WithMaxTokens(200),
		)

		if err != nil {
			// If summarization fails, keep the original result
			summarizedResults = append(summarizedResults, result)
			continue
		}

		summary := strings.TrimSpace(responseContent.String())

		// Drop irrelevant content
		if strings.ToUpper(summary) == "IRRELEVANT" {
			continue
		}

		// Create new summarized result
		summarizedResult := &ToolResult{
			Sentences:    []string{summary},
			Attributions: result.Attributions, // Preserve attributions
			Title:        result.Title,
			Metadata:     make(map[string]interface{}),
			Success:      true,
		}

		// Copy metadata and add summarization info
		for k, v := range result.Metadata {
			summarizedResult.Metadata[k] = v
		}
		summarizedResult.Metadata["summarized"] = true
		summarizedResult.Metadata["original_sentence_count"] = len(result.Sentences)

		summarizedResults = append(summarizedResults, summarizedResult)
	}

	return summarizedResults, nil
}

// formatToolResult formats a ToolResult into a human-readable string for prompts
func (a *Agent) formatToolResult(toolName string, result *ToolResult) string {
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

	if len(result.Attributions) > 0 {
		formatted.WriteString("\nSources:")
		for _, attribution := range result.Attributions {
			formatted.WriteString(fmt.Sprintf("\n- %s", attribution))
		}
	}

	if len(result.Metadata) > 0 {
		formatted.WriteString("\nAdditional Info:")
		for key, value := range result.Metadata {
			formatted.WriteString(fmt.Sprintf("\n- %s: %v", key, value))
		}
	}

	return formatted.String()
}

// NewToolResult creates a new successful ToolResult
func NewToolResult(title string, sentences []string) *ToolResult {
	return &ToolResult{
		Title:     title,
		Sentences: sentences,
		Success:   true,
		Metadata:  make(map[string]interface{}),
	}
}

// NewSingleToolResult creates a slice with a single ToolResult for simple tools
func NewSingleToolResult(title string, sentences []string) []*ToolResult {
	return []*ToolResult{NewToolResult(title, sentences)}
}

// NewToolResultWithAttribution creates a ToolResult with attribution
func NewToolResultWithAttribution(title string, sentences []string, attributions []string) *ToolResult {
	return &ToolResult{
		Title:        title,
		Sentences:    sentences,
		Attributions: attributions,
		Success:      true,
		Metadata:     make(map[string]interface{}),
	}
}

// NewToolResultError creates a failed ToolResult
func NewToolResultError(errorMsg string) *ToolResult {
	return &ToolResult{
		Success: false,
		Error:   errorMsg,
	}
}

// NewSingleToolResultError creates a slice with a single error ToolResult
func NewSingleToolResultError(errorMsg string) []*ToolResult {
	return []*ToolResult{NewToolResultError(errorMsg)}
}

// NewMathToolResult creates a ToolResult specifically for mathematical calculations
func NewMathToolResult(expression string, result string, steps []string) []*ToolResult {
	sentences := []string{fmt.Sprintf("%s = %s", expression, result)}
	if len(steps) > 0 {
		sentences = append(sentences, "Calculation steps:")
		sentences = append(sentences, steps...)
	}

	toolResult := NewToolResult("Mathematical Calculation", sentences)
	toolResult.Metadata["expression"] = expression
	toolResult.Metadata["result"] = result
	toolResult.Metadata["calculation_type"] = "arithmetic"

	return []*ToolResult{toolResult}
}

// NewDateTimeToolResult creates a ToolResult for date/time operations
func NewDateTimeToolResult(operation string, result string, timezone string) []*ToolResult {
	sentences := []string{fmt.Sprintf("%s: %s", operation, result)}

	toolResult := NewToolResult("Date/Time Operation", sentences)
	toolResult.Metadata["operation"] = operation
	toolResult.Metadata["timezone"] = timezone
	toolResult.Metadata["timestamp"] = time.Now().Unix()

	return []*ToolResult{toolResult}
}

// NewSearchToolResult creates a ToolResult for search operations
func NewSearchToolResult(query string, results []string, sources []string) []*ToolResult {
	sentences := make([]string, 0, len(results)+1)
	sentences = append(sentences, fmt.Sprintf("Search results for: %s", query))
	sentences = append(sentences, results...)

	toolResult := NewToolResultWithAttribution("Search Results", sentences, sources)
	toolResult.Metadata["query"] = query
	toolResult.Metadata["result_count"] = len(results)

	return []*ToolResult{toolResult}
}

// NewMultiSearchToolResult creates multiple ToolResults for search operations with separate sources
func NewMultiSearchToolResult(query string, resultPairs []SearchResultPair) []*ToolResult {
	results := make([]*ToolResult, 0, len(resultPairs))

	for i, pair := range resultPairs {
		sentences := []string{pair.Content}
		attributions := []string{pair.Source}

		toolResult := NewToolResultWithAttribution(
			fmt.Sprintf("Search Result %d", i+1),
			sentences,
			attributions,
		)
		toolResult.Metadata["query"] = query
		toolResult.Metadata["result_index"] = i
		toolResult.Metadata["source"] = pair.Source

		results = append(results, toolResult)
	}

	return results
}

// SearchResultPair represents a search result with its content and source
type SearchResultPair struct {
	Content string
	Source  string
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
