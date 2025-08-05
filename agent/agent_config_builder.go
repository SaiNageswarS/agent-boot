package agent

import "github.com/SaiNageswarS/agent-boot/llm"

type AgentConfigBuilder struct {
	config AgentConfig
}

func NewAgentConfigBuilder() *AgentConfigBuilder {
	return &AgentConfigBuilder{}
}

func (b *AgentConfigBuilder) WithMiniModel(client llm.LLMClient, model string) *AgentConfigBuilder {
	b.config.MiniModel.Client = client
	b.config.MiniModel.Model = model
	return b
}

func (b *AgentConfigBuilder) WithBigModel(client llm.LLMClient, model string) *AgentConfigBuilder {
	b.config.BigModel.Client = client
	b.config.BigModel.Model = model
	return b
}

func (b *AgentConfigBuilder) AddTool(tool MCPTool) *AgentConfigBuilder {
	b.config.Tools = append(b.config.Tools, tool)
	return b
}

func (b *AgentConfigBuilder) WithPrompt(prompt PromptTemplate) *AgentConfigBuilder {
	b.config.Prompt = prompt
	return b
}

func (b *AgentConfigBuilder) WithMaxTokens(max int) *AgentConfigBuilder {
	b.config.MaxTokens = max
	return b
}

func (b *AgentConfigBuilder) Build() AgentConfig {
	return b.config
}
