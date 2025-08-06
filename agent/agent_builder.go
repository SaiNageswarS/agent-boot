package agent

import "github.com/SaiNageswarS/agent-boot/llm"

type AgentBuilder struct {
	config AgentConfig
}

func NewAgentBuilder() *AgentBuilder {
	return &AgentBuilder{}
}

func (b *AgentBuilder) WithMiniModel(client llm.LLMClient) *AgentBuilder {
	b.config.MiniModel = client
	return b
}

func (b *AgentBuilder) WithBigModel(client llm.LLMClient) *AgentBuilder {
	b.config.BigModel = client
	return b
}

func (b *AgentBuilder) AddTool(tool MCPTool) *AgentBuilder {
	b.config.Tools = append(b.config.Tools, tool)
	return b
}

func (b *AgentBuilder) WithPrompt(prompt PromptTemplate) *AgentBuilder {
	b.config.Prompt = prompt
	return b
}

func (b *AgentBuilder) WithMaxTokens(max int) *AgentBuilder {
	b.config.MaxTokens = max
	return b
}

func (b *AgentBuilder) Build() *Agent {
	return &Agent{config: b.config}
}
