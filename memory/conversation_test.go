package memory

import (
	"context"
	"testing"

	"github.com/SaiNageswarS/agent-boot/llm"
	"github.com/stretchr/testify/assert"
)

func TestConversationManager_LoadSession(t *testing.T) {
	t.Run("nil collection", func(t *testing.T) {
		cm := NewConversationManager(nil, 10)
		conversation := cm.LoadSession(context.Background(), "test-session")

		assert.NotNil(t, conversation)
		assert.Empty(t, conversation.Messages)
	})
}

func TestConversation_AddMessages(t *testing.T) {
	t.Run("AddUserMessage", func(t *testing.T) {
		conversation := &Conversation{}
		conversation.AddUserMessage("Hello")

		assert.Len(t, conversation.Messages, 1)
		assert.Equal(t, "user", conversation.Messages[0].Role)
		assert.Equal(t, "Hello", conversation.Messages[0].Content)
		assert.False(t, conversation.Messages[0].IsToolResult)
	})

	t.Run("AddAssistantMessage", func(t *testing.T) {
		conversation := &Conversation{}
		conversation.AddAssistantMessage("Hi there!")

		assert.Len(t, conversation.Messages, 1)
		assert.Equal(t, "assistant", conversation.Messages[0].Role)
		assert.Equal(t, "Hi there!", conversation.Messages[0].Content)
	})

	t.Run("AddToolResult", func(t *testing.T) {
		conversation := &Conversation{}
		conversation.AddToolResult("Tool executed successfully")

		assert.Len(t, conversation.Messages, 1)
		assert.Equal(t, "user", conversation.Messages[0].Role)
		assert.Equal(t, "Tool executed successfully", conversation.Messages[0].Content)
		assert.True(t, conversation.Messages[0].IsToolResult)
	})
}

func TestConversationManager_trimForSession(t *testing.T) {
	tests := []struct {
		name     string
		maxMsgs  int
		input    []llm.Message
		expected []llm.Message
	}{
		{
			name:     "empty messages",
			maxMsgs:  5,
			input:    []llm.Message{},
			expected: []llm.Message{},
		},
		{
			name:    "maxMsgs is 0",
			maxMsgs: 0,
			input: []llm.Message{
				{Role: "user", Content: "Hello"},
			},
			expected: []llm.Message{},
		},
		{
			name:    "fewer messages than max",
			maxMsgs: 5,
			input: []llm.Message{
				{Role: "user", Content: "Hello"},
				{Role: "assistant", Content: "Hi!"},
			},
			expected: []llm.Message{
				{Role: "user", Content: "Hello"},
				{Role: "assistant", Content: "Hi!"},
			},
		},
		{
			name:    "exactly max messages",
			maxMsgs: 2,
			input: []llm.Message{
				{Role: "user", Content: "Hello"},
				{Role: "assistant", Content: "Hi!"},
				{Role: "user", Content: "How are you?"},
				{Role: "assistant", Content: "I'm good!"},
			},
			expected: []llm.Message{
				{Role: "user", Content: "Hello"},
				{Role: "assistant", Content: "Hi!"},
				{Role: "user", Content: "How are you?"},
				{Role: "assistant", Content: "I'm good!"},
			},
		},
		{
			name:    "more messages than max",
			maxMsgs: 2,
			input: []llm.Message{
				{Role: "user", Content: "Hello"},
				{Role: "assistant", Content: "Hi!"},
				{Role: "user", Content: "How are you?"},
				{Role: "assistant", Content: "I'm good!"},
				{Role: "user", Content: "What's the weather?"},
				{Role: "assistant", Content: "It's sunny!"},
			},
			expected: []llm.Message{
				{Role: "user", Content: "How are you?"},
				{Role: "assistant", Content: "I'm good!"},
				{Role: "user", Content: "What's the weather?"},
				{Role: "assistant", Content: "It's sunny!"},
			},
		},
		{
			name:    "tool results should not count as user messages",
			maxMsgs: 1,
			input: []llm.Message{
				{Role: "user", Content: "Hello"},
				{Role: "assistant", Content: "Hi!"},
				{Role: "user", Content: "Tool result", IsToolResult: true},
				{Role: "user", Content: "How are you?"},
			},
			expected: []llm.Message{
				{Role: "user", Content: "How are you?"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cm := NewConversationManager(nil, tt.maxMsgs)
			result := cm.trimForSession(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestConversationManager_SaveSession(t *testing.T) {
	t.Run("nil collection", func(t *testing.T) {
		cm := NewConversationManager(nil, 10)
		conversation := &Conversation{
			ID:       "test-session",
			Messages: []llm.Message{{Role: "user", Content: "Hello"}},
		}

		err := cm.SaveSession(context.Background(), conversation)
		assert.NoError(t, err) // Should not error with nil collection
	})
}
