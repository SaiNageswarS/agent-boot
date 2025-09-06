package llm

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/ollama/ollama/api"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewGroqClient(t *testing.T) {
	// Test with missing API key
	withEnv("GROQ_API_KEY", "", func(logger *MockLogger) {
		NewGroqClient("llama-3.3-70b-versatile")
		assert.True(t, logger.isFatalCalled)
	})

	// Test with API key set
	withEnv("GROQ_API_KEY", "test-key", func(logger *MockLogger) {
		client := NewGroqClient("llama-3.3-70b-versatile")
		assert.NotNil(t, client)
		assert.Equal(t, "llama-3.3-70b-versatile", client.GetModel())
	})
}

func TestGroqClientCapabilities(t *testing.T) {
	withEnv("GROQ_API_KEY", "test-key", func(logger *MockLogger) {
		tests := []struct {
			model        string
			capabilities Capability
		}{
			{"llama-3.3-70b-versatile", NativeToolCalling},
			{"llama-3.1-8b-instant", NativeToolCalling},
			{"openai/gpt-oss-20b", NativeToolCalling},
			{"openai/gpt-oss-120b", NativeToolCalling},
			{"meta-llama/llama-4-scout-17b-16e-instruct", NativeToolCalling},
			{"some-unsupported-model", Capability(0)},
		}

		for _, tt := range tests {
			t.Run(tt.model, func(t *testing.T) {
				client := NewGroqClient(tt.model)
				assert.Equal(t, tt.capabilities, client.Capabilities())
			})
		}
	})
}

func TestGroqClientGenerateInference(t *testing.T) {
	// Create a mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "/openai/v1/chat/completions", r.URL.Path)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
		assert.Equal(t, "Bearer test-key", r.Header.Get("Authorization"))

		// Mock response
		response := groqResponse{
			Choices: []groqChoice{
				{
					Message: groqMessage{
						Content: "Hello, this is a test response",
					},
				},
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	withEnv("GROQ_API_KEY", "test-key", func(logger *MockLogger) {
		client := NewGroqClient("llama-3.3-70b-versatile").(*GroqClient)
		client.url = server.URL + "/openai/v1/chat/completions"

		messages := []Message{
			{Role: "user", Content: "Hello"},
		}

		var result string
		err := client.GenerateInference(context.Background(), messages, func(chunk string) error {
			result = chunk
			return nil
		})

		require.NoError(t, err)
		assert.Equal(t, "Hello, this is a test response", result)
	})
}

func TestGroqClientGenerateInferenceWithTools(t *testing.T) {
	// Create a mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Mock response with tool calls
		response := groqResponse{
			Choices: []groqChoice{
				{
					Message: groqMessage{
						ToolCalls: []groqToolCall{
							{
								ID:   "call_123",
								Type: "function",
								Function: groqToolCallFunction{
									Name:      "calculator",
									Arguments: `{"expression": "2+2"}`,
								},
							},
						},
					},
				},
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	os.Setenv("GROQ_API_KEY", "test-key")
	defer os.Unsetenv("GROQ_API_KEY")

	client := NewGroqClient("llama-3.3-70b-versatile").(*GroqClient)
	client.url = server.URL + "/openai/v1/chat/completions"

	messages := []Message{
		{Role: "user", Content: "Calculate 2+2"},
	}

	tools := []api.Tool{
		{
			Function: api.ToolFunction{
				Name:        "calculator",
				Description: "Calculate mathematical expressions",
			},
		},
	}

	var toolCalls []api.ToolCall
	err := client.GenerateInferenceWithTools(
		context.Background(),
		messages,
		func(chunk string) error {
			return nil
		},
		func(calls []api.ToolCall) error {
			toolCalls = calls
			return nil
		},
		WithTools(tools),
	)

	require.NoError(t, err)
	require.Len(t, toolCalls, 1)
	assert.Equal(t, "calculator", toolCalls[0].Function.Name)
	assert.Equal(t, "2+2", toolCalls[0].Function.Arguments["expression"])
}

func TestConvertToolsToGroqFormat(t *testing.T) {
	tools := []api.Tool{
		{
			Function: api.ToolFunction{
				Name:        "calculator",
				Description: "Calculate mathematical expressions",
			},
		},
	}

	groqTools := convertToolsToGroqFormat(tools)

	require.Len(t, groqTools, 1)
	assert.Equal(t, "function", groqTools[0].Type)
	assert.Equal(t, "calculator", groqTools[0].Function.Name)
	assert.Equal(t, "Calculate mathematical expressions", groqTools[0].Function.Description)
}

func TestGroqClientWithSystemPrompt(t *testing.T) {
	// Create a mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var request groqRequest
		err := json.NewDecoder(r.Body).Decode(&request)
		require.NoError(t, err)

		// Check that system message was added
		require.Len(t, request.Messages, 2)
		assert.Equal(t, "system", request.Messages[0].Role)
		assert.Equal(t, "You are a helpful assistant", request.Messages[0].Content)
		assert.Equal(t, "user", request.Messages[1].Role)
		assert.Equal(t, "Hello", request.Messages[1].Content)

		// Mock response
		response := groqResponse{
			Choices: []groqChoice{
				{
					Message: groqMessage{
						Content: "Hello! How can I help you?",
					},
				},
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	os.Setenv("GROQ_API_KEY", "test-key")
	defer os.Unsetenv("GROQ_API_KEY")

	client := NewGroqClient("llama-3.3-70b-versatile").(*GroqClient)
	client.url = server.URL + "/openai/v1/chat/completions"

	messages := []Message{
		{Role: "user", Content: "Hello"},
	}

	var result string
	err := client.GenerateInference(context.Background(), messages, func(chunk string) error {
		result = chunk
		return nil
	}, WithSystemPrompt("You are a helpful assistant"))

	require.NoError(t, err)
	assert.Equal(t, "Hello! How can I help you?", result)
}
