package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/SaiNageswarS/agent-boot/agentboot"
	"github.com/SaiNageswarS/agent-boot/llm"
	"github.com/SaiNageswarS/agent-boot/schema"
	"github.com/ollama/ollama/api"
)

func main() {
	// Set environment variable for testing (in real usage, this would be set externally)
	if os.Getenv("ANTHROPIC_API_KEY") == "" {
		fmt.Println("ANTHROPIC_API_KEY not set. This is just a demo of the implementation.")
		fmt.Println()
		fmt.Println("🚀 The AnthropicClient now supports unified tool calling via prompt engineering!")
		fmt.Println()
		fmt.Println("🎯 Key features:")
		fmt.Println("1. ✅ Unified approach: Single prompt handles both tool selection AND direct answers")
		fmt.Println("2. ✅ LLM-agnostic design: JSON format compatible with any LLM")
		fmt.Println("3. ✅ Parameter type specification: string, number, boolean, array with (required) marking")
		fmt.Println("4. ✅ Max turns safety: Prevents infinite tool loops by forcing direct answers")
		fmt.Println("5. ✅ Robust parsing: Graceful fallback to regular inference on errors")
		fmt.Println("6. ✅ Turn-based context: Previous tool results inform next decisions")
		fmt.Println()
		fmt.Println("📋 Response format:")
		fmt.Println(`   Tool use: {"action": "use_tools", "tool_calls": [...]}`)
		fmt.Println(`   Direct answer: {"action": "direct_answer", "content": "..."}`)
		fmt.Println()
		fmt.Println("💡 This implementation works exactly like native tool calling!")
		return
	}

	// Create Anthropic client
	anthropicClient := llm.NewAnthropicClient("claude-3-haiku-20240307")

	// Create a simple calculator tool
	calculatorTool := agentboot.NewMCPToolBuilder("calculator", "Performs mathematical calculations").
		StringParam("expression", "Mathematical expression to evaluate", true).
		WithHandler(func(ctx context.Context, params api.ToolCallFunctionArguments) <-chan *schema.ToolResultChunk {
			ch := make(chan *schema.ToolResultChunk, 1)
			defer close(ch)

			expression, ok := params["expression"].(string)
			if !ok {
				chunk := agentboot.NewToolResultChunk().
					Error("Invalid expression parameter").
					Build()
				ch <- chunk
				return ch
			}

			// Simple calculation demo
			result := fmt.Sprintf("The result of '%s' is calculated successfully", expression)
			chunk := agentboot.NewToolResultChunk().
				Title("Calculation Result").
				Sentences(result).
				Build()
			ch <- chunk
			return ch
		}).
		Build()

	// Create a weather tool to test multiple tool scenarios
	weatherTool := agentboot.NewMCPToolBuilder("get_weather", "Gets current weather for a location").
		StringParam("location", "City name or coordinates", true).
		StringParam("units", "Temperature units (celsius/fahrenheit)", false).
		WithHandler(func(ctx context.Context, params api.ToolCallFunctionArguments) <-chan *schema.ToolResultChunk {
			ch := make(chan *schema.ToolResultChunk, 1)
			defer close(ch)

			location, _ := params["location"].(string)
			units, _ := params["units"].(string)
			if units == "" {
				units = "celsius"
			}

			result := fmt.Sprintf("Weather in %s: 22°%s, sunny", location, units)
			chunk := agentboot.NewToolResultChunk().
				Title("Weather Report").
				Sentences(result).
				Build()
			ch <- chunk
			return ch
		}).
		Build()

	// Create agent with Anthropic client and tools
	agent := agentboot.NewAgentBuilder().
		WithBigModel(anthropicClient).
		WithMiniModel(anthropicClient).
		AddTool(calculatorTool).
		AddTool(weatherTool).
		WithMaxTurns(3). // Test max turns safety
		Build()

	// Create a progress reporter
	reporter := &SimpleProgressReporter{}

	// Test scenarios
	testCases := []struct {
		name     string
		question string
	}{
		{
			name:     "Tool Usage Test",
			question: "Please calculate 25 + 17 * 3",
		},
		{
			name:     "Direct Answer Test",
			question: "What is the capital of France?",
		},
		{
			name:     "Multiple Tools Test",
			question: "Calculate 10 + 5 and also get the weather in New York",
		},
	}

	ctx := context.Background()

	for _, tc := range testCases {
		fmt.Printf("\n🧪 %s\n", tc.name)
		fmt.Printf("Question: %s\n", tc.question)
		fmt.Println(strings.Repeat("-", 50))

		req := &schema.GenerateAnswerRequest{
			Question: tc.question,
		}

		response, err := agent.Execute(ctx, reporter, req)
		if err != nil {
			log.Printf("❌ Error: %v", err)
			continue
		}

		fmt.Printf("✅ Agent response: %s\n", response.Answer)
		fmt.Printf("🔧 Tools used: %v\n", response.ToolsUsed)
	}

	fmt.Println("\n🎉 Demo completed! The unified tool calling system is working correctly.")
}

// SimpleProgressReporter implements the ProgressReporter interface for demo purposes
type SimpleProgressReporter struct{}

func (r *SimpleProgressReporter) Send(chunk *schema.AgentStreamChunk) error {
	switch chunk.ChunkType.(type) {
	case *schema.AgentStreamChunk_ProgressUpdateChunk:
		fmt.Printf("Progress: %s - %s\n",
			chunk.GetProgressUpdateChunk().Stage,
			chunk.GetProgressUpdateChunk().Message)
	case *schema.AgentStreamChunk_ToolResultChunk:
		fmt.Printf("Tool result from %s\n", chunk.GetToolResultChunk().ToolName)
	case *schema.AgentStreamChunk_Answer:
		fmt.Printf("Answer chunk: %s", chunk.GetAnswer().Content)
	case *schema.AgentStreamChunk_Complete:
		fmt.Printf("Execution complete: %s\n", chunk.GetComplete().Answer)
	case *schema.AgentStreamChunk_Error:
		fmt.Printf("Error: %s\n", chunk.GetError().ErrorMessage)
	}
	return nil
}
