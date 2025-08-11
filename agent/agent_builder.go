package agent

import "github.com/SaiNageswarS/agent-boot/llm"

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

func (b *AgentBuilder) Build() *Agent {
	return &Agent{config: b.config}
}
