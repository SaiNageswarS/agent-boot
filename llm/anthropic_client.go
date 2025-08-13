package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/SaiNageswarS/agent-boot/prompts"
	"github.com/SaiNageswarS/go-api-boot/logger"
	"github.com/ollama/ollama/api"
)

type AnthropicClient struct {
	apiKey     string
	httpClient *http.Client
	url        string
	model      string
}

func NewAnthropicClient(model string) LLMClient {
	apiKey := os.Getenv("ANTHROPIC_API_KEY")
	if apiKey == "" {
		// Providers are designed for dependency injection.
		// If the API key is not set, we log a fatal error.
		logger.Fatal("ANTHROPIC_API_KEY environment variable is not set")
		return nil // This will never be reached, but it's good practice to return nil here.
	}

	return &AnthropicClient{
		apiKey:     apiKey,
		httpClient: &http.Client{},
		url:        "https://api.anthropic.com/v1/messages",
		model:      model,
	}
}

func (c *AnthropicClient) Capabilities() Capability {
	return 0 // Anthropic does not support native tool calling
}

func (c *AnthropicClient) GetModel() string {
	return c.model
}

func (c *AnthropicClient) GenerateInference(ctx context.Context, messages []Message, callback func(chunk string) error, opts ...LLMOption) error {
	settings := LLMSettings{
		model:       c.model,
		temperature: 0.7,
		maxTokens:   4096,
	}

	// Apply options
	for _, opt := range opts {
		opt(&settings)
	}

	request := anthropicRequest{
		Model:       settings.model,
		MaxTokens:   settings.maxTokens,
		Temperature: settings.temperature,
		System:      settings.system,
		Messages:    messages,
	}

	jsonData, err := json.Marshal(request)
	if err != nil {
		return fmt.Errorf("error marshaling request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", c.url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("error creating request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", c.apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("error making request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("error reading response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var response anthropicResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return fmt.Errorf("error unmarshaling response: %w", err)
	}

	if len(response.Content) == 0 {
		return fmt.Errorf("no content in response")
	}

	return callback(response.Content[0].Text)
}

func (c *AnthropicClient) GenerateInferenceWithTools(
	ctx context.Context,
	messages []Message,
	contentCallback func(chunk string) error,
	toolCallback func(toolCalls []api.ToolCall) error,
	opts ...LLMOption,
) error {
	settings := LLMSettings{
		model:       c.model,
		temperature: 0.7,
		maxTokens:   4096,
	}

	// Apply options
	for _, opt := range opts {
		opt(&settings)
	}

	// If no tools are provided, use regular inference
	if len(settings.tools) == 0 {
		return c.GenerateInference(ctx, messages, contentCallback, opts...)
	}

	// Use unified inference approach
	return c.unifiedInferenceWithTools(ctx, messages, contentCallback, toolCallback, settings.tools, 1, 3, "")
}

// unifiedInferenceWithTools handles both tool calling and direct answers in a single unified approach
func (c *AnthropicClient) unifiedInferenceWithTools(
	ctx context.Context,
	messages []Message,
	contentCallback func(chunk string) error,
	toolCallback func(toolCalls []api.ToolCall) error,
	tools []api.Tool,
	currentTurn int,
	maxTurns int,
	previousToolResults string,
) error {
	// Create tool descriptions for the prompt
	toolDescriptions := make([]string, len(tools))
	for i, tool := range tools {
		// Format: "tool_name: description (parameters: param1:type, param2:type, ...)"
		params := []string{}
		if tool.Function.Parameters.Properties != nil {
			for paramName, paramProp := range tool.Function.Parameters.Properties {
				paramType := "string" // default
				if len(paramProp.Type) > 0 {
					paramType = string(paramProp.Type[0])
				}

				// Check if parameter is required
				isRequired := false
				for _, req := range tool.Function.Parameters.Required {
					if req == paramName {
						isRequired = true
						break
					}
				}

				paramStr := fmt.Sprintf("%s:%s", paramName, paramType)
				if isRequired {
					paramStr += " (required)"
				}
				params = append(params, paramStr)
			}
		}

		var paramStr string
		if len(params) > 0 {
			paramStr = fmt.Sprintf(" (parameters: %s)", strings.Join(params, ", "))
		}

		toolDescriptions[i] = fmt.Sprintf("%s: %s%s",
			tool.Function.Name,
			tool.Function.Description,
			paramStr)
	}

	// Get the last user message as the query
	var query string
	for i := len(messages) - 1; i >= 0; i-- {
		if messages[i].Role == "user" {
			query = messages[i].Content
			break
		}
	}

	// Force direct answer if we've reached max turns
	forceDirectAnswer := currentTurn >= maxTurns

	// Create unified inference prompt data
	promptData := prompts.InferenceWithToolPromptData{
		ToolDescriptions:    toolDescriptions,
		MaxTools:            len(tools),
		Query:               query,
		Context:             "", // Could be enhanced to include conversation context
		CurrentTurn:         currentTurn,
		MaxTurns:            maxTurns,
		PreviousToolResults: previousToolResults,
	}

	// Render the unified inference prompt
	systemPrompt, userPrompt, err := prompts.RenderInferenceWithToolPrompt(promptData)
	if err != nil {
		return fmt.Errorf("error rendering unified inference prompt: %w", err)
	}

	// If we need to force a direct answer, modify the system prompt
	if forceDirectAnswer {
		systemPrompt += "\n\n**IMPORTANT: You have reached the maximum number of turns. You MUST provide a direct answer now, do not use any more tools.**"
	}

	// Create messages for unified inference
	inferenceMessages := []Message{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: userPrompt},
	}

	// Get unified inference response
	var inferenceResponse strings.Builder
	err = c.GenerateInference(ctx, inferenceMessages,
		func(chunk string) error {
			inferenceResponse.WriteString(chunk)
			return nil
		},
		WithMaxTokens(4096))

	if err != nil {
		return fmt.Errorf("error getting unified inference: %w", err)
	}

	// Parse the unified response
	return c.parseUnifiedResponse(inferenceResponse.String(), contentCallback, toolCallback, forceDirectAnswer)
}

// parseUnifiedResponse parses the unified response and calls appropriate callbacks
func (c *AnthropicClient) parseUnifiedResponse(
	response string,
	contentCallback func(chunk string) error,
	toolCallback func(toolCalls []api.ToolCall) error,
	forceDirectAnswer bool,
) error {
	// Clean the response to extract JSON
	response = strings.TrimSpace(response)

	// Find JSON content within the response
	startIdx := strings.Index(response, "{")
	endIdx := strings.LastIndex(response, "}")

	if startIdx == -1 || endIdx == -1 || startIdx >= endIdx {
		// If we can't parse JSON and it's forced direct answer, treat as content
		if forceDirectAnswer {
			return contentCallback(response)
		}
		return fmt.Errorf("no valid JSON found in response")
	}

	jsonStr := response[startIdx : endIdx+1]

	var unifiedResponse unifiedInferenceResponse
	if err := json.Unmarshal([]byte(jsonStr), &unifiedResponse); err != nil {
		// If we can't parse JSON and it's forced direct answer, treat as content
		if forceDirectAnswer {
			return contentCallback(response)
		}
		return fmt.Errorf("error unmarshaling unified response: %w", err)
	}

	// Handle the response based on action
	switch unifiedResponse.Action {
	case "direct_answer":
		if unifiedResponse.Content == "" {
			return fmt.Errorf("direct_answer action but no content provided")
		}
		return contentCallback(unifiedResponse.Content)

	case "use_tools":
		if len(unifiedResponse.ToolCalls) == 0 {
			// If no tool calls but action is use_tools, fall back to direct answer if forced
			if forceDirectAnswer {
				return contentCallback("I apologize, but I couldn't determine the appropriate tools to use for your request.")
			}
			return fmt.Errorf("use_tools action but no tool calls provided")
		}

		// Convert to api.ToolCall format
		toolCalls := make([]api.ToolCall, len(unifiedResponse.ToolCalls))
		for i, tc := range unifiedResponse.ToolCalls {
			toolCalls[i] = api.ToolCall{
				Function: api.ToolCallFunction{
					Name:      tc.Function.Name,
					Arguments: tc.Function.Arguments,
				},
			}
		}

		return toolCallback(toolCalls)

	default:
		// If action is unknown and it's forced direct answer, treat as content
		if forceDirectAnswer {
			return contentCallback(response)
		}
		return fmt.Errorf("unknown action: %s", unifiedResponse.Action)
	}
}

type anthropicRequest struct {
	Model       string    `json:"model"`
	MaxTokens   int       `json:"max_tokens"`
	Messages    []Message `json:"messages"`
	System      string    `json:"system,omitempty"`
	Temperature float64   `json:"temperature"`
}

// anthropicResponse represents the response from Anthropic API
type anthropicResponse struct {
	Content []content `json:"content"`
	ID      string    `json:"id"`
	Model   string    `json:"model"`
	Role    string    `json:"role"`
	Type    string    `json:"type"`
}

// content represents the content in the response
type content struct {
	Text string `json:"text"`
	Type string `json:"type"`
}

// unifiedInferenceResponse represents the unified response structure
type unifiedInferenceResponse struct {
	Action    string `json:"action"` // "use_tools" or "direct_answer"
	Content   string `json:"content,omitempty"`
	ToolCalls []struct {
		Function struct {
			Name      string                 `json:"name"`
			Arguments map[string]interface{} `json:"arguments"`
		} `json:"function"`
		Reasoning string `json:"reasoning"`
	} `json:"tool_calls,omitempty"`
}
