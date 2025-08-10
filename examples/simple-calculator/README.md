# Simple Calculator Agent Example

This example demonstrates a basic agent with a calculator tool that can perform mathematical operations.

## Features

- Simple calculator tool that can evaluate basic mathematical expressions
- Console progress reporting with real-time updates
- Turn-based conversation with the agent
- Tool execution with result streaming

## Prerequisites

1. **Go 1.24+** installed
2. **Ollama** running locally with a model available
   ```bash
   # Install Ollama (macOS/Linux)
   curl -fsSL https://ollama.ai/install.sh | sh
   
   # Pull a model
   ollama pull llama3.2:latest
   
   # Start Ollama (if not running as service)
   ollama serve
   ```

## Setup

1. Make sure you're in the project root:
   ```bash
   cd /path/to/agent-boot
   ```

2. Set the OLLAMA_HOST environment variable:
   ```bash
   export OLLAMA_HOST="http://localhost:11434"
   ```

## Running the Example

```bash
go run examples/simple-calculator/main.go
```

## Expected Output

```
🤖 Starting calculation...
📋 Progress: Running tool calculator with arguments: map[expression:15*23+7]
🔧 Tool Result: Mathematical Calculation
   Evaluating: 15*23+7
   Result: 352
💭 Thinking: Looking at this math problem, I need to calculate 15 multiplied by 23, and then add 7 to the result.

Let me break this down step by step:

First: 15 × 23 = 345
Then: 345 + 7 = 352

Therefore, 15 multiplied by 23, plus 7, equals 352.
🎉 Task completed!

✅ Final Answer: Looking at this math problem, I need to calculate 15 multiplied by 23, and then add 7 to the result.

Let me break this down step by step:

First: 15 × 23 = 345
Then: 345 + 7 = 352

Therefore, 15 multiplied by 23, plus 7, equals 352.
⏱️  Processing Time: 1247ms
🔧 Tools Used: [calculator]
```

## How It Works

1. **Agent Setup**: Creates an agent with Ollama LLM client and a calculator tool
2. **Tool Definition**: Calculator tool accepts mathematical expressions as string parameters
3. **Tool Handler**: Evaluates the expression and returns a structured result
4. **Progress Reporting**: Console reporter shows real-time progress updates
5. **Execution**: Agent processes the question, calls tools as needed, and provides a final answer

## Customization

### Adding More Operations

Extend the `evaluateExpression` function to support more mathematical operations:

```go
func evaluateExpression(expr string) (string, error) {
    // Use a proper math expression parser like:
    // - github.com/Knetic/govaluate
    // - github.com/antonmedv/expr
    // - Custom parser for your specific needs
}
```

### Using Different Models

```go
// Use a different Ollama model
llmClient := llm.NewOllamaClient("mistral:latest")

// Or use Anthropic (requires API key)
llmClient := llm.NewAnthropicClient("claude-3-5-sonnet-20241022")
```

### Adding More Tools

```go
// Add a unit converter tool
converterTool := agent.NewMCPTool("unit_converter", "Converts between different units").
    StringParam("value", "Value to convert", true).
    StringParam("from_unit", "Source unit", true).
    StringParam("to_unit", "Target unit", true).
    WithHandler(converterHandler).
    Build()

agent := agent.NewAgentBuilder().
    WithBigModel(llmClient).
    AddTool(calculatorTool).
    AddTool(converterTool).
    Build()
```

## Troubleshooting

### "OLLAMA_HOST environment variable is not set"
Set the environment variable:
```bash
export OLLAMA_HOST="http://localhost:11434"
```

### "connection refused"
Make sure Ollama is running:
```bash
ollama serve
```

### "model not found"
Pull the required model:
```bash
ollama pull llama3.2:latest
```
