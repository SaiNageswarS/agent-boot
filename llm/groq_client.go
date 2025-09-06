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

	"github.com/SaiNageswarS/go-api-boot/logger"
	"github.com/ollama/ollama/api"
)

type GroqClient struct {
	apiKey     string
	httpClient *http.Client
	url        string
	model      string
}

func NewGroqClient(model string) LLMClient {
	apiKey := os.Getenv("GROQ_API_KEY")
	if apiKey == "" {
		logger.Fatal("GROQ_API_KEY environment variable is not set")
		return nil
	}

	return &GroqClient{
		apiKey:     apiKey,
		httpClient: &http.Client{},
		url:        "https://api.groq.com/openai/v1/chat/completions",
		model:      model,
	}
}

func (c *GroqClient) Capabilities() Capability {
	// Models that support tool calling based on Groq documentation
	toolSupportedModels := []string{
		"llama-3.3-70b-versatile",
		"llama-3.1-8b-instant",
		"openai/gpt-oss-20b",
		"openai/gpt-oss-120b",
		"meta-llama/llama-4-scout-17b-16e-instruct",
		"meta-llama/llama-4-maverick-17b-128e-instruct",
		"moonshotai/kimi-k2-instruct",
		"moonshotai/kimi-k2-instruct-0905",
	}

	for _, supportedModel := range toolSupportedModels {
		if strings.Contains(c.model, supportedModel) || c.model == supportedModel {
			return NativeToolCalling
		}
	}

	return 0
}

func (c *GroqClient) GetModel() string {
	return c.model
}

func (c *GroqClient) GenerateInference(ctx context.Context, messages []Message, callback func(chunk string) error, opts ...LLMOption) error {
	// Default settings
	settings := LLMSettings{
		model:       c.model,
		temperature: 0.7,
		maxTokens:   4096,
		stream:      false,
	}

	// Apply options
	for _, opt := range opts {
		opt(&settings)
	}

	request := groqRequest{
		Model:       settings.model,
		Messages:    messages,
		Temperature: settings.temperature,
		MaxTokens:   settings.maxTokens,
		Stream:      settings.stream,
	}

	// Add system prompt if provided (Groq uses system message in messages array)
	if settings.system != "" {
		systemMsg := Message{
			Role:    "system",
			Content: settings.system,
		}
		request.Messages = append([]Message{systemMsg}, request.Messages...)
	}

	return c.makeRequest(ctx, request, callback, nil)
}

func (c *GroqClient) GenerateInferenceWithTools(
	ctx context.Context,
	messages []Message,
	contentCallback func(chunk string) error,
	toolCallback func(toolCalls []api.ToolCall) error,
	opts ...LLMOption,
) error {
	// Default settings
	settings := LLMSettings{
		model:       c.model,
		temperature: 0.7,
		maxTokens:   4096,
		stream:      false,
	}

	// Apply options
	for _, opt := range opts {
		opt(&settings)
	}

	request := groqRequest{
		Model:       settings.model,
		Messages:    messages,
		Temperature: settings.temperature,
		MaxTokens:   settings.maxTokens,
		Stream:      settings.stream,
		Tools:       convertToolsToGroqFormat(settings.tools),
		ToolChoice:  "auto",
	}

	// Add system prompt if provided
	if settings.system != "" {
		systemMsg := Message{
			Role:    "system",
			Content: settings.system,
		}
		request.Messages = append([]Message{systemMsg}, request.Messages...)
	}

	return c.makeRequest(ctx, request, contentCallback, toolCallback)
}

func (c *GroqClient) makeRequest(
	ctx context.Context,
	request groqRequest,
	contentCallback func(chunk string) error,
	toolCallback func(toolCalls []api.ToolCall) error,
) error {
	jsonData, err := json.Marshal(request)
	if err != nil {
		return fmt.Errorf("error marshaling request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", c.url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("error creating request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

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

	var response groqResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return fmt.Errorf("error unmarshaling response: %w", err)
	}

	if len(response.Choices) == 0 {
		return fmt.Errorf("no choices in response")
	}

	choice := response.Choices[0]

	// Handle tool calls
	if len(choice.Message.ToolCalls) > 0 && toolCallback != nil {
		// Convert Groq tool calls to Ollama format for compatibility
		ollamaToolCalls := make([]api.ToolCall, len(choice.Message.ToolCalls))
		for i, tc := range choice.Message.ToolCalls {
			// Parse the JSON arguments string into a map
			var args map[string]any
			if err := json.Unmarshal([]byte(tc.Function.Arguments), &args); err != nil {
				return fmt.Errorf("error parsing tool call arguments: %w", err)
			}

			ollamaToolCalls[i] = api.ToolCall{
				Function: api.ToolCallFunction{
					Name:      tc.Function.Name,
					Arguments: args,
				},
			}
		}
		return toolCallback(ollamaToolCalls)
	}

	// Handle regular content
	if choice.Message.Content != "" && contentCallback != nil {
		return contentCallback(choice.Message.Content)
	}

	return nil
}

// convertToolsToGroqFormat converts Ollama tools to Groq format
func convertToolsToGroqFormat(tools []api.Tool) []groqTool {
	if len(tools) == 0 {
		return nil
	}

	groqTools := make([]groqTool, len(tools))
	for i, tool := range tools {
		groqTools[i] = groqTool{
			Type: "function",
			Function: groqFunction{
				Name:        tool.Function.Name,
				Description: tool.Function.Description,
				Parameters:  tool.Function.Parameters,
			},
		}
	}
	return groqTools
}

// Groq API types
type groqRequest struct {
	Model       string     `json:"model"`
	Messages    []Message  `json:"messages"`
	Temperature float64    `json:"temperature,omitempty"`
	MaxTokens   int        `json:"max_completion_tokens,omitempty"`
	Stream      bool       `json:"stream,omitempty"`
	Tools       []groqTool `json:"tools,omitempty"`
	ToolChoice  string     `json:"tool_choice,omitempty"`
}

type groqTool struct {
	Type     string       `json:"type"`
	Function groqFunction `json:"function"`
}

type groqFunction struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	Parameters  interface{} `json:"parameters"`
}

type groqResponse struct {
	ID      string       `json:"id"`
	Object  string       `json:"object"`
	Created int64        `json:"created"`
	Model   string       `json:"model"`
	Choices []groqChoice `json:"choices"`
	Usage   groqUsage    `json:"usage"`
}

type groqChoice struct {
	Index        int         `json:"index"`
	Message      groqMessage `json:"message"`
	FinishReason string      `json:"finish_reason"`
}

type groqMessage struct {
	Role      string         `json:"role"`
	Content   string         `json:"content"`
	ToolCalls []groqToolCall `json:"tool_calls,omitempty"`
}

type groqToolCall struct {
	ID       string               `json:"id"`
	Type     string               `json:"type"`
	Function groqToolCallFunction `json:"function"`
}

type groqToolCallFunction struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

type groqUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}
