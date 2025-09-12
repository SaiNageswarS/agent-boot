package agentboot

import (
	"github.com/SaiNageswarS/agent-boot/llm"
	"github.com/SaiNageswarS/agent-boot/memory"
	"github.com/SaiNageswarS/go-api-boot/odm"
)

type AgentBuilder struct {
	config AgentConfig
}

func NewAgentBuilder() *AgentBuilder {
	return &AgentBuilder{
		config: AgentConfig{
			MaxTurns:  5,
			MaxTokens: 2000,
		},
	}
}

func (b *AgentBuilder) WithMiniModel(client llm.LLMClient) *AgentBuilder {
	b.config.MiniModel = client
	return b
}

func (b *AgentBuilder) WithBigModel(client llm.LLMClient) *AgentBuilder {
	b.config.BigModel = client
	return b
}

func (b *AgentBuilder) WithToolSelector(client llm.LLMClient) *AgentBuilder {
	b.config.ToolSelector = client
	return b
}

func (b *AgentBuilder) WithSystemPrompt(prompt string) *AgentBuilder {
	b.config.SystemPrompt = prompt
	return b
}

func (b *AgentBuilder) AddTool(tool MCPTool) *AgentBuilder {
	b.config.Tools = append(b.config.Tools, tool)
	return b
}

func (b *AgentBuilder) WithMaxTokens(max int) *AgentBuilder {
	b.config.MaxTokens = max
	return b
}

func (b *AgentBuilder) WithMaxTurns(maxTurns int) *AgentBuilder {
	b.config.MaxTurns = maxTurns
	return b
}

func (b *AgentBuilder) WithConversationManager(collection odm.OdmCollectionInterface[memory.Conversation], maxMsgs int) *AgentBuilder {
	b.config.ConversationManager = memory.NewConversationManager(collection, maxMsgs)
	return b
}

// Deprecated: Use WithConversationManager instead
func (b *AgentBuilder) WithMaxSessionMessages(max int) *AgentBuilder {
	if b.config.ConversationManager != nil {
		b.config.ConversationManager.SetMaxMessages(max)
	}
	return b
}

func (b *AgentBuilder) Build() *Agent {
	if b.config.ToolSelector == nil {
		b.config.ToolSelector = llm.NewOllamaClient("gpt-oss:20b") // Default tool selector
	}

	return &Agent{config: b.config}
}
