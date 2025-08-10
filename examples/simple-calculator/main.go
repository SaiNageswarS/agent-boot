package main

import (
	"context"
	"fmt"
	"log"
	"strconv"

	"github.com/SaiNageswarS/agent-boot/agent"
	"github.com/SaiNageswarS/agent-boot/llm"
	"github.com/SaiNageswarS/agent-boot/schema"
	"github.com/ollama/ollama/api"
)

func main() {
	// Create LLM client (requires Ollama running locally)
	llmClient := llm.NewOllamaClient("gpt-oss:20b")

	// Create a simple calculator tool
	calculatorTool := agent.NewMCPTool("calculator", "Performs basic mathematical calculations").
		StringParam("expression", "Mathematical expression to evaluate (e.g., '2+2', '10*5')", true).
		WithHandler(calculatorHandler).
		Build()

	// Build the agent
	agentInstance := agent.NewAgentBuilder().
		WithBigModel(llmClient).
		WithMiniModel(llmClient).
		AddTool(calculatorTool).
		WithMaxTokens(2000).
		WithMaxTurns(5).
		Build()

	// Create a simple progress reporter that logs to console
	reporter := &ConsoleReporter{}

	// Execute the agent
	ctx := context.Background()
	request := &schema.GenerateAnswerRequest{
		Question: "What is 15 multiplied by 23, and then add 7 to the result?",
		Context:  "Please solve this math problem step by step using the calculator tool.",
	}

	fmt.Println("🤖 Starting calculation...")
	response, err := agentInstance.Execute(ctx, reporter, request)
	if err != nil {
		log.Fatalf("Error executing agent: %v", err)
	}

	fmt.Printf("\n✅ Final Answer: %s\n", response.Answer)
	fmt.Printf("⏱️  Processing Time: %dms\n", response.ProcessingTime)
	fmt.Printf("🔧 Tools Used: %v\n", response.ToolsUsed)
}

// calculatorHandler implements a simple calculator
func calculatorHandler(ctx context.Context, params api.ToolCallFunctionArguments) <-chan *schema.ToolResultChunk {
	ch := make(chan *schema.ToolResultChunk, 1)

	go func() {
		defer close(ch)

		expression, ok := params["expression"].(string)
		if !ok {
			chunk := agent.NewToolResultChunk().
				Error("Invalid expression parameter").
				Build()
			ch <- chunk
			return
		}

		// Simple calculator implementation (you would use a proper math parser in production)
		result, err := evaluateExpression(expression)
		if err != nil {
			chunk := agent.NewToolResultChunk().
				Error(fmt.Sprintf("Calculation error: %v", err)).
				Build()
			ch <- chunk
			return
		}

		// Create a successful result chunk
		chunk := agent.NewMathToolResult(expression, result, []string{
			fmt.Sprintf("Evaluating: %s", expression),
			fmt.Sprintf("Result: %s", result),
		})

		ch <- chunk
	}()

	return ch
}

// Simple expression evaluator (very basic - you'd want a proper parser for production)
func evaluateExpression(expr string) (string, error) {
	// This is a very simplified calculator for demo purposes
	// In production, you'd use a proper math expression parser

	switch expr {
	case "15*23":
		return "345", nil
	case "15 * 23":
		return "345", nil
	case "345+7":
		return "352", nil
	case "345 + 7":
		return "352", nil
	case "15*23+7":
		return "352", nil
	case "15 * 23 + 7":
		return "352", nil
	default:
		// Fallback for simple addition
		if len(expr) >= 3 && expr[1] == '+' {
			a, err1 := strconv.Atoi(string(expr[0]))
			c, err2 := strconv.Atoi(string(expr[2]))
			if err1 == nil && err2 == nil {
				return strconv.Itoa(a + c), nil
			}
		}
		return "", fmt.Errorf("unsupported expression: %s", expr)
	}
}

// ConsoleReporter logs progress to the console
type ConsoleReporter struct{}

func (r *ConsoleReporter) Send(event *schema.AgentStreamChunk) error {
	switch chunk := event.ChunkType.(type) {
	case *schema.AgentStreamChunk_ProgressUpdateChunk:
		fmt.Printf("📋 Progress: %s\n", chunk.ProgressUpdateChunk.Message)
	case *schema.AgentStreamChunk_ToolResultChunk:
		if chunk.ToolResultChunk.Error != "" {
			fmt.Printf("❌ Tool Error: %s\n", chunk.ToolResultChunk.Error)
		} else {
			fmt.Printf("🔧 Tool Result: %s\n", chunk.ToolResultChunk.Title)
			for _, sentence := range chunk.ToolResultChunk.Sentences {
				fmt.Printf("   %s\n", sentence)
			}
		}
	case *schema.AgentStreamChunk_Answer:
		fmt.Printf("💭 Thinking: %s", chunk.Answer.Content)
	case *schema.AgentStreamChunk_Complete:
		fmt.Printf("🎉 Task completed!\n")
	case *schema.AgentStreamChunk_Error:
		fmt.Printf("❌ Error: %s\n", chunk.Error.ErrorMessage)
	}
	return nil
}
