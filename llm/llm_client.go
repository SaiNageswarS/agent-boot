package llm

import (
	"context"

	"github.com/ollama/ollama/api"
)

type Capability uint8

const (
	NativeToolCalling Capability = 1 << iota
)

type LLMClient interface {
	GenerateInference(
		ctx context.Context,
		messages []Message,
		callback func(chunk string) error,
		opts ...LLMOption,
	) error

	// GenerateInferenceWithTools supports native tool calling
	GenerateInferenceWithTools(
		ctx context.Context,
		messages []Message,
		contentCallback func(chunk string) error,
		toolCallback func(toolCalls []api.ToolCall) error,
		opts ...LLMOption,
	) error

	Capabilities() Capability

	GetModel() string
}

type LLMSettings struct {
	model       string     // model name
	temperature float64    // randomness (0.0 to 1.0)
	maxTokens   int        // maximum tokens to generate
	system      string     // system prompt
	stream      bool       // whether to stream response
	tools       []api.Tool // tools to use for tool calling
}

type LLMOption func(*LLMSettings)

// Common options for all LLM providers
func WithTemperature(temp float64) LLMOption {
	return func(s *LLMSettings) { s.temperature = temp }
}

func WithMaxTokens(tokens int) LLMOption {
	return func(s *LLMSettings) { s.maxTokens = tokens }
}

func WithSystemPrompt(prompt string) LLMOption {
	return func(s *LLMSettings) { s.system = prompt }
}

func WithStreaming(stream bool) LLMOption {
	return func(s *LLMSettings) { s.stream = stream }
}

func WithTools(tools []api.Tool) LLMOption {
	return func(s *LLMSettings) { s.tools = tools }
}

type Message struct {
	Role    string `json:"role"`    // "user", "assistant", "system"
	Content string `json:"content"` // the message content
}
