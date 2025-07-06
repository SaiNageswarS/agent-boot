package agent

import "github.com/SaiNageswarS/go-api-boot/llm"

type AgentFlow struct {
	llmClient llm.LLMClient
	model     string
}

func New(llmClient llm.LLMClient, model string) *AgentFlow {
	return &AgentFlow{
		llmClient: llmClient,
		model:     model,
	}
}
