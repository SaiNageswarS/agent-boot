package memory

import (
	"github.com/SaiNageswarS/agent-boot/llm"
)

// Conversation represents a conversation session with messages
type Conversation struct {
	ID       string        `bson:"_id"`
	Messages []llm.Message `bson:"messages"`
}

func (m Conversation) Id() string {
	return m.ID
}

func (m Conversation) CollectionName() string {
	return "conversations"
}

func (m *Conversation) AddUserMessage(content string) {
	m.Messages = append(m.Messages, llm.Message{Role: "user", Content: content})
}

func (m *Conversation) AddAssistantMessage(content string) {
	m.Messages = append(m.Messages, llm.Message{Role: "assistant", Content: content})
}

func (m *Conversation) AddToolResult(content string) {
	m.Messages = append(m.Messages, llm.Message{Role: "user", Content: content, IsToolResult: true})
}
