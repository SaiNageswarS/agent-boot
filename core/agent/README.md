# Agent Boot - Intelligent Agent System

This package provides a sophisticated agent system that integrates with LLM clients to provide intelligent tool selection and answer generation capabilities.

## Features

- **Dual Model Architecture**: Uses both mini and big models for optimal performance and cost efficiency
- **MCP Tool Integration**: Support for Model Context Protocol (MCP) tools with intelligent selection
- **Embedded Template System**: Markdown-based templates compiled into the binary with Go's embed
- **Template Manager**: Centralized template loading and rendering with type-safe data structures
- **Intelligent Tool Selection**: AI-powered tool selection based on query analysis
- **Robust Parsing**: Supports both structured text and JSON formats for LLM responses
- **Small Model Friendly**: Uses non-JSON output format that works better with smaller models
- **Context-Aware Responses**: Considers context and tool results for comprehensive answers
- **Modular Prompt Management**: Separated prompt templates for better maintainability
- **Extensible Architecture**: Easy to add new templates and customize prompts
- **Backward Compatibility**: Supports legacy JSON format for existing integrations

## Architecture

The agent system consists of several key components:

### Core Components

1. **Agent**: The main orchestrator that coordinates between models, tools, and prompts
2. **MCPTool**: Represents executable tools with handlers
3. **PromptTemplate**: Reusable templates for generating prompts
4. **AgentConfig**: Configuration for the agent including models and tools
5. **Template System**: Embedded Markdown templates with Go templating (`templates/` directory)
6. **TemplateManager**: Centralized template loading and rendering system

### Embedded Template System

The system now uses embedded Markdown templates stored in the `templates/` directory:

- **Embedded Files**: Templates are compiled into the binary using `//go:embed`
- **Markdown Format**: Templates use Markdown for better readability and structure
- **Go Templates**: Support for variables, conditionals, and loops
- **Type Safety**: Structured data models for template rendering
- **Hot Reloadable**: During development, templates can be modified without recompilation (when not embedded)

### LLM Response Formats

The system supports multiple output formats to accommodate different model capabilities:

#### Structured Text Format (Primary)
```
TOOL_SELECTION_START

TOOL: calculator
CONFIDENCE: 0.95
REASONING: User asked to calculate an expression
PARAMETERS:
  expression: 15 + 25 * 2

TOOL: weather
CONFIDENCE: 0.8
REASONING: User requested weather information
PARAMETERS:
  location: New York
  units: metric

TOOL_SELECTION_END
```

#### JSON Format (Legacy Support)
```json
[
  {
    "tool_name": "calculator",
    "confidence": 0.95,
    "reasoning": "User asked to calculate an expression",
    "parameters": {
      "expression": "15 + 25 * 2"
    }
  }
]
```

The structured text format is preferred as it's more reliable with smaller models that may struggle with JSON formatting.

### Prompt Template Usage

```go
// Example of using the template system
data := ToolSelectionPromptData{
    ToolDescriptions: []string{
        "- calculator: Performs arithmetic calculations",
        "- weather: Gets weather information",
    },
    MaxTools: 2,
    Query:     "What's 2+2 and the weather?",
    Context:   "User planning activity",
}

systemPrompt, userPrompt, err := RenderToolSelectionPrompt(data)
if err != nil {
    log.Fatal(err)
}
```

### Model Selection Strategy

The agent automatically chooses between mini and big models based on:
- Query complexity (length > 100 characters)
- Number of tool results to synthesize
- Presence of complex keywords (analyze, compare, summarize, etc.)

## Usage

### Basic Setup with Embedded Templates

```go
package main

import (
    "context"
    "github.com/SaiNageswarS/agent-boot/core/agent"
    "github.com/SaiNageswarS/agent-boot/core/llm"
)

func main() {
    // Create LLM clients
    miniClient := llm.ProvideAnthropicClient() // or other provider
    bigClient := llm.ProvideAnthropicClient()  // or other provider

    // Configure the agent
    config := agent.AgentConfig{
        MiniModel: struct {
            Client llm.LLMClient
            Model  string
        }{
            Client: miniClient,
            Model:  "claude-3-haiku-20240307",
        },
        BigModel: struct {
            Client llm.LLMClient
            Model  string
        }{
            Client: bigClient,
            Model:  "claude-3-opus-20240229",
        },
        Tools:     []agent.MCPTool{},
        MaxTokens: 2000,
    }

    // Create the agent with embedded templates
    agentInstance, err := agent.NewAgentWithTemplates(config)
    if err != nil {
        log.Fatal(err)
    }
    
    // The agent now has access to embedded Markdown templates
}
```

### Template Manager Usage

```go
// Create a custom template manager
tm, err := agent.NewTemplateManager()
if err != nil {
    log.Fatal(err)
}

// Render individual templates
data := agent.PromptData{
    Query:   "Analyze market trends",
    Context: "Q4 financial analysis",
    ToolResults: []string{
        "Revenue increased 15%",
        "Costs decreased 5%",
    },
}

analysisPrompt, err := tm.RenderTemplate("analysis_prompt", data)
if err != nil {
    log.Fatal(err)
}

fmt.Println(analysisPrompt)
```

### Adding Tools

```go
// Create a calculator tool
calculatorTool := agent.MCPTool{
    Name:        "calculator",
    Description: "Performs basic arithmetic calculations",
    Parameters: map[string]interface{}{
        "expression": "string",
    },
    Handler: func(ctx context.Context, params map[string]interface{}) (interface{}, error) {
        expression := params["expression"].(string)
        // Implement calculation logic
        result := calculate(expression)
        return map[string]interface{}{
            "result": result,
        }, nil
    },
}

// Add tool to agent
agentInstance.AddTool(calculatorTool)
```

### Adding Prompt Templates

```go
// Create a prompt template
analysisTemplate := agent.PromptTemplate{
    Name:     "analysis",
    Template: "Analyze the following query: {{query}}\n\nContext: {{context}}\n\nTool Results: {{tool_results}}\n\nProvide a detailed analysis.",
    Variables: []string{"query", "context", "tool_results"},
    Metadata: map[string]string{
        "type": "analysis",
    },
}

// Add prompt to agent
agentInstance.AddPrompt("analysis", analysisTemplate)
```

### Generating Answers

```go
// Simple query without tools
request := agent.GenerateAnswerRequest{
    Query:    "What is machine learning?",
    UseTools: false,
}

response, err := agentInstance.GenerateAnswer(context.Background(), request)
if err != nil {
    log.Fatal(err)
}

fmt.Println("Answer:", response.Answer)
fmt.Println("Model Used:", response.ModelUsed)
```

### Complex Query with Tools

```go
// Complex query with tool usage
request := agent.GenerateAnswerRequest{
    Query:          "Calculate 15 * 25 and explain the mathematical concept",
    Context:        "User is learning basic arithmetic",
    PromptTemplate: "analysis",
    UseTools:       true,
    MaxIterations:  3,
}

response, err := agentInstance.GenerateAnswer(context.Background(), request)
if err != nil {
    log.Fatal(err)
}

fmt.Println("Answer:", response.Answer)
fmt.Println("Tools Used:", len(response.ToolsUsed))
for _, tool := range response.ToolsUsed {
    fmt.Printf("- %s (confidence: %.2f)\n", tool.Tool.Name, tool.Confidence)
}
```

### Tool Selection Only

```go
// Select tools without generating full answer
toolRequest := agent.ToolSelectionRequest{
    Query:    "I need to calculate something and check the weather",
    Context:  "Planning an outdoor event",
    MaxTools: 2,
}

tools, err := agentInstance.SelectTools(context.Background(), toolRequest)
if err != nil {
    log.Fatal(err)
}

for _, tool := range tools {
    fmt.Printf("Selected: %s - %s\n", tool.Tool.Name, tool.Reasoning)
}
```

## API Reference

### Files Structure

- **`agent.go`**: Main agent implementation with core logic
- **`prompts.go`**: Template-based prompt rendering system (legacy support)
- **`template_manager.go`**: Centralized template management with embedded files
- **`templates/`**: Directory containing embedded Markdown templates
  - **`tool_selection_system.md`**: System prompt for tool selection
  - **`tool_selection_user.md`**: User prompt for tool selection
  - **`analysis_prompt.md`**: Structured analysis template
  - **`default_answer.md`**: General answer generation template
  - **`README.md`**: Template documentation
- **`prompts_test.go`**: Tests for the prompt template system (legacy)
- **`template_manager_test.go`**: Tests for the template manager
- **`integration_test.go`**: Integration tests demonstrating full functionality
- **`agent_example_test.go`**: Usage examples and basic tests

### Types

#### TemplateManager
Centralized manager for loading and rendering embedded Markdown templates.

#### PromptData
Generic data structure for template rendering with support for queries, context, and tool results.

#### ToolSelectionPromptData
Specialized data structure for tool selection prompt templates.

#### MCPTool
Represents a Model Context Protocol tool with a handler function.

#### PromptTemplate
A reusable template with variable substitution support (legacy).

#### AgentConfig
Configuration structure for initializing an agent, now with optional TemplateManager.

#### GenerateAnswerRequest
Request structure for answer generation.

#### GenerateAnswerResponse
Response structure containing the generated answer and metadata.

### Functions

#### NewTemplateManager() (*TemplateManager, error)
Creates a new template manager with all embedded templates loaded.

#### NewAgentWithTemplates(config AgentConfig) (*Agent, error)
Creates a new agent instance with embedded templates automatically configured.

#### RenderToolSelectionPrompt(data ToolSelectionPromptData) (systemPrompt, userPrompt string, err error)
Renders tool selection prompts using embedded templates (legacy support).

#### RenderToolSelectionPromptWithManager(tm *TemplateManager, data ToolSelectionPromptData) (systemPrompt, userPrompt string, err error)
Renders tool selection prompts using the template manager.

#### parseToolSelections(response string) ([]ToolSelection, error)
Parses LLM responses in either structured text or JSON format to extract tool selections.

#### parseStructuredTextSelections(response string) ([]ToolSelection, error)
Parses the structured text format with TOOL_SELECTION_START/END markers.

#### parseJSONSelections(response string) ([]ToolSelection, error)
Parses the legacy JSON format for backward compatibility.

### Methods

#### NewAgent(config AgentConfig) *Agent
Creates a new agent instance with the provided configuration.

#### SelectTools(ctx context.Context, req ToolSelectionRequest) ([]ToolSelection, error)
Selects appropriate tools for a given query using the mini model.

#### GenerateAnswer(ctx context.Context, req GenerateAnswerRequest) (*GenerateAnswerResponse, error)
Main API for generating answers with optional tool usage.

#### AddTool(tool MCPTool)
Adds a new tool to the agent's toolkit.

#### AddPrompt(name string, template PromptTemplate)
Adds a new prompt template to the agent.

## Best Practices

1. **Tool Design**: Keep tools focused and single-purpose
2. **Prompt Templates**: Use templates for consistent formatting
3. **Error Handling**: Always handle errors from tool execution
4. **Context Management**: Provide relevant context for better results
5. **Model Selection**: Let the agent choose the appropriate model automatically

## Examples

See `agent_example_test.go` for comprehensive usage examples and test cases.
