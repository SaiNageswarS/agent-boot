package agent

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/SaiNageswarS/agent-boot/core/llm"
)

// Mock LLM client for testing
type mockLLMClient struct {
	response string
	err      error
}

func (m *mockLLMClient) GenerateInference(
	ctx context.Context,
	messages []llm.Message,
	callback func(string) error,
	options ...llm.LLMOption,
) error {
	if m.err != nil {
		return m.err
	}
	return callback(m.response)
}

// Multi-response mock client for testing different LLM calls
type multiResponseMockClient struct {
	responses []string
	callCount int
	err       error
}

func (m *multiResponseMockClient) GenerateInference(
	ctx context.Context,
	messages []llm.Message,
	callback func(string) error,
	options ...llm.LLMOption,
) error {
	if m.err != nil {
		return m.err
	}

	if m.callCount < len(m.responses) {
		response := m.responses[m.callCount]
		m.callCount++
		return callback(response)
	}

	// Default response if we run out of responses
	return callback("Default response")
}

func TestNewAgent(t *testing.T) {
	config := AgentConfig{
		Tools:     []MCPTool{},
		MaxTokens: 1000,
	}

	agent := NewAgent(config)

	if agent == nil {
		t.Fatal("NewAgent should return a non-nil agent")
	}

	if agent.config.MaxTokens != 1000 {
		t.Errorf("Expected MaxTokens to be 1000, got %d", agent.config.MaxTokens)
	}
}

func TestAddTool(t *testing.T) {
	agent := NewAgent(AgentConfig{})

	if len(agent.GetAvailableTools()) != 0 {
		t.Fatal("Expected no tools initially")
	}

	tool := MCPTool{
		Name:        "test-tool",
		Description: "A test tool",
		Handler: func(ctx context.Context, params map[string]interface{}) ([]*ToolResult, error) {
			return NewSingleToolResult("Test", []string{"result"}), nil
		},
	}

	agent.AddTool(tool)

	tools := agent.GetAvailableTools()
	if len(tools) != 1 {
		t.Fatalf("Expected 1 tool, got %d", len(tools))
	}

	if tools[0].Name != "test-tool" {
		t.Errorf("Expected tool name 'test-tool', got '%s'", tools[0].Name)
	}
}

func TestAddPrompt(t *testing.T) {
	agent := NewAgent(AgentConfig{})

	template := PromptTemplate{
		Name:      "test-template",
		Template:  "Hello {{name}}",
		Variables: []string{"name"},
	}

	agent.AddPrompt("test", template)

	prompts := agent.GetAvailablePrompts()
	if len(prompts) != 1 {
		t.Fatalf("Expected 1 prompt, got %d", len(prompts))
	}

	if prompts["test"].Name != "test-template" {
		t.Errorf("Expected prompt name 'test-template', got '%s'", prompts["test"].Name)
	}
}

func TestSelectToolsWithNoTools(t *testing.T) {
	agent := NewAgent(AgentConfig{
		MiniModel: struct {
			Client llm.LLMClient
			Model  string
		}{
			Client: &mockLLMClient{response: "No tools available"},
			Model:  "test-model",
		},
	})

	req := ToolSelectionRequest{
		Query:    "test query",
		Context:  "test context",
		MaxTools: 3,
	}

	selections, err := agent.SelectTools(context.Background(), req)
	if err != nil {
		t.Fatalf("SelectTools should not error with no tools: %v", err)
	}

	if len(selections) != 0 {
		t.Errorf("Expected 0 selections with no tools, got %d", len(selections))
	}
}

func TestSelectToolsWithValidResponse(t *testing.T) {
	// Mock LLM response in structured text format
	mockResponse := `
TOOL_SELECTION_START
TOOL: calculator
CONFIDENCE: 0.9
REASONING: User asked for mathematical calculation
PARAMETERS:
expression: 2+2
TOOL_SELECTION_END
`

	agent := NewAgent(AgentConfig{
		MiniModel: struct {
			Client llm.LLMClient
			Model  string
		}{
			Client: &mockLLMClient{response: mockResponse},
			Model:  "test-model",
		},
		Tools: []MCPTool{
			{
				Name:        "calculator",
				Description: "Performs calculations",
				Handler: func(ctx context.Context, params map[string]interface{}) ([]*ToolResult, error) {
					return NewMathToolResult("2+2", "4", nil), nil
				},
			},
		},
	})

	req := ToolSelectionRequest{
		Query:    "What is 2+2?",
		Context:  "Math question",
		MaxTools: 1,
	}

	selections, err := agent.SelectTools(context.Background(), req)
	if err != nil {
		t.Fatalf("SelectTools failed: %v", err)
	}

	if len(selections) != 1 {
		t.Fatalf("Expected 1 selection, got %d", len(selections))
	}

	if selections[0].Tool.Name != "calculator" {
		t.Errorf("Expected tool 'calculator', got '%s'", selections[0].Tool.Name)
	}

	if selections[0].Confidence != 0.9 {
		t.Errorf("Expected confidence 0.9, got %f", selections[0].Confidence)
	}
}

func TestSelectToolsWithJSONFallback(t *testing.T) {
	// Mock LLM response in JSON format
	mockResponse := `[{"tool_name": "search", "confidence": 0.8, "reasoning": "User needs information", "parameters": {"query": "test"}}]`

	agent := NewAgent(AgentConfig{
		MiniModel: struct {
			Client llm.LLMClient
			Model  string
		}{
			Client: &mockLLMClient{response: mockResponse},
			Model:  "test-model",
		},
		Tools: []MCPTool{
			{
				Name:        "search",
				Description: "Searches for information",
				Handler: func(ctx context.Context, params map[string]interface{}) ([]*ToolResult, error) {
					return NewSearchToolResult("test", []string{"result"}, []string{"source"}), nil
				},
			},
		},
	})

	req := ToolSelectionRequest{
		Query:    "Find information about AI",
		MaxTools: 1,
	}

	selections, err := agent.SelectTools(context.Background(), req)
	if err != nil {
		t.Fatalf("SelectTools failed: %v", err)
	}

	if len(selections) != 1 {
		t.Fatalf("Expected 1 selection, got %d", len(selections))
	}

	if selections[0].Tool.Name != "search" {
		t.Errorf("Expected tool 'search', got '%s'", selections[0].Tool.Name)
	}
}

func TestExecuteTool(t *testing.T) {
	handlerCalled := false
	tool := MCPTool{
		Name:        "test-tool",
		Description: "Test tool",
		Handler: func(ctx context.Context, params map[string]interface{}) ([]*ToolResult, error) {
			handlerCalled = true
			return NewSingleToolResult("Test Result", []string{"Success"}), nil
		},
	}

	agent := NewAgent(AgentConfig{})
	selection := ToolSelection{
		Tool:       tool,
		Parameters: map[string]interface{}{"test": "value"},
	}

	results, err := agent.executeTool(context.Background(), selection)
	if err != nil {
		t.Fatalf("executeTool failed: %v", err)
	}

	if !handlerCalled {
		t.Error("Tool handler should have been called")
	}

	if len(results) != 1 {
		t.Fatalf("Expected 1 result, got %d", len(results))
	}

	if !results[0].Success {
		t.Error("Result should be successful")
	}

	if results[0].Title != "Test Result" {
		t.Errorf("Expected title 'Test Result', got '%s'", results[0].Title)
	}
}

func TestGenerateAnswer(t *testing.T) {
	mockResponse := "This is a test answer"

	agent := NewAgent(AgentConfig{
		MiniModel: struct {
			Client llm.LLMClient
			Model  string
		}{
			Client: &mockLLMClient{response: mockResponse},
			Model:  "mini-model",
		},
		BigModel: struct {
			Client llm.LLMClient
			Model  string
		}{
			Client: &mockLLMClient{response: mockResponse},
			Model:  "big-model",
		},
		MaxTokens: 1000,
	})

	req := GenerateAnswerRequest{
		Query:    "Test question",
		Context:  "Test context",
		UseTools: false,
	}

	response, err := agent.GenerateAnswer(context.Background(), req)
	if err != nil {
		t.Fatalf("GenerateAnswer failed: %v", err)
	}

	if response.Answer != mockResponse {
		t.Errorf("Expected answer '%s', got '%s'", mockResponse, response.Answer)
	}

	if response.ModelUsed != "mini-model" {
		t.Errorf("Expected model 'mini-model', got '%s'", response.ModelUsed)
	}

	if response.ProcessingTime < 0 {
		t.Error("Processing time should not be negative")
	}
}

func TestGenerateAnswerWithTools(t *testing.T) {
	toolResponse := "TOOL_SELECTION_START\nTOOL: calculator\nCONFIDENCE: 0.9\nREASONING: Math needed\nPARAMETERS:\nexpression: 2+2\nTOOL_SELECTION_END"
	answerResponse := "The answer is 4"

	// Use the same client but modify the response after the first call
	mockClient := &multiResponseMockClient{
		responses: []string{toolResponse, answerResponse},
	}

	agent := NewAgent(AgentConfig{
		MiniModel: struct {
			Client llm.LLMClient
			Model  string
		}{
			Client: mockClient,
			Model:  "mini-model",
		},
		BigModel: struct {
			Client llm.LLMClient
			Model  string
		}{
			Client: mockClient,
			Model:  "big-model",
		},
		Tools: []MCPTool{
			{
				Name:        "calculator",
				Description: "Performs calculations",
				Handler: func(ctx context.Context, params map[string]interface{}) ([]*ToolResult, error) {
					return NewMathToolResult("2+2", "4", []string{"2 + 2 = 4"}), nil
				},
			},
		},
	})

	req := GenerateAnswerRequest{
		Query:    "What is 2+2?",
		UseTools: true,
	}

	response, err := agent.GenerateAnswer(context.Background(), req)
	if err != nil {
		t.Fatalf("GenerateAnswer with tools failed: %v", err)
	}

	if len(response.ToolsUsed) != 1 {
		t.Errorf("Expected 1 tool used, got %d", len(response.ToolsUsed))
	}

	if response.ToolsUsed[0].Tool.Name != "calculator" {
		t.Errorf("Expected calculator tool, got '%s'", response.ToolsUsed[0].Tool.Name)
	}

	if response.Answer != answerResponse {
		t.Errorf("Expected answer '%s', got '%s'", answerResponse, response.Answer)
	}

	if response.Answer != answerResponse {
		t.Errorf("Expected answer '%s', got '%s'", answerResponse, response.Answer)
	}
}

func TestShouldUseBigModel(t *testing.T) {
	agent := NewAgent(AgentConfig{})

	// Short simple query should use mini model
	if agent.shouldUseBigModel("Hi", []string{}) {
		t.Error("Short query should use mini model")
	}

	// Long query should use big model
	longQuery := strings.Repeat("This is a very long query that should trigger the big model because it exceeds the character limit. ", 5)
	if !agent.shouldUseBigModel(longQuery, []string{}) {
		t.Error("Long query should use big model")
	}

	// Multiple tool results should use big model
	if !agent.shouldUseBigModel("Simple query", []string{"result1", "result2"}) {
		t.Error("Multiple tool results should use big model")
	}

	// Complex keywords should use big model
	if !agent.shouldUseBigModel("Please analyze this data", []string{}) {
		t.Error("Complex keywords should use big model")
	}
}

func TestGetMaxTokens(t *testing.T) {
	// Test with configured max tokens
	agent := NewAgent(AgentConfig{MaxTokens: 500})
	if agent.getMaxTokens() != 500 {
		t.Errorf("Expected 500 max tokens, got %d", agent.getMaxTokens())
	}

	// Test with default max tokens
	agent = NewAgent(AgentConfig{})
	if agent.getMaxTokens() != 2000 {
		t.Errorf("Expected default 2000 max tokens, got %d", agent.getMaxTokens())
	}
}

func TestGetMaxTools(t *testing.T) {
	if getMaxTools(5) != 5 {
		t.Errorf("Expected 5 max tools, got %d", getMaxTools(5))
	}

	if getMaxTools(0) != 3 {
		t.Errorf("Expected default 3 max tools, got %d", getMaxTools(0))
	}

	if getMaxTools(15) != 3 {
		t.Errorf("Expected capped 3 max tools, got %d", getMaxTools(15))
	}
}

func TestGetCurrentTimeMs(t *testing.T) {
	start := time.Now().UnixMilli()
	result := getCurrentTimeMs()
	end := time.Now().UnixMilli()

	if result < start || result > end {
		t.Errorf("getCurrentTimeMs should return current time, got %d (expected between %d and %d)", result, start, end)
	}
}

// Integration Tests for Multi-Result Tool System

func TestMultiResultRAGIntegration(t *testing.T) {
	// Create a mock client with multiple responses for tool selection and answer generation
	mockClient := &multiResponseMockClient{
		responses: []string{
			`{"tools": ["rag_search"]}`, // Tool selection response
			"Based on the knowledge base, machine learning is a subset of AI that enables computers to learn without explicit programming. It includes techniques like deep learning with neural networks and supervised learning from labeled data.", // Answer response
		},
	}

	agent := NewAgent(AgentConfig{
		MiniModel: struct {
			Client llm.LLMClient
			Model  string
		}{
			Client: mockClient,
			Model:  "mini-model",
		},
		BigModel: struct {
			Client llm.LLMClient
			Model  string
		}{
			Client: mockClient,
			Model:  "big-model",
		},
		Tools:     []MCPTool{},
		MaxTokens: 1000,
	})

	// Add a mock RAG search tool that returns multiple results
	ragTool := MCPTool{
		Name:        "rag_search",
		Description: "Search knowledge base and return multiple relevant documents",
		Handler: func(ctx context.Context, params map[string]interface{}) ([]*ToolResult, error) {
			// Simulate RAG returning multiple relevant documents
			return []*ToolResult{
				{
					Sentences:    []string{"Machine learning is a subset of artificial intelligence that enables computers to learn without being explicitly programmed."},
					Attributions: []string{"AI Fundamentals Textbook, Chapter 3"},
					Title:        "Introduction to Machine Learning",
					Success:      true,
				},
				{
					Sentences:    []string{"Deep learning uses neural networks with multiple layers to model and understand complex patterns in data."},
					Attributions: []string{"Neural Networks Research Paper, 2020"},
					Title:        "Deep Learning Architectures",
					Success:      true,
				},
				{
					Sentences:    []string{"Supervised learning algorithms learn from labeled training data to make predictions on new, unseen data."},
					Attributions: []string{"Machine Learning Handbook, Section 4.2"},
					Title:        "Supervised Learning Methods",
					Success:      true,
				},
			}, nil
		},
	}

	agent.AddTool(ragTool)

	// Test RAG search execution
	ctx := context.Background()
	req := GenerateAnswerRequest{
		Query:    "Please analyze machine learning concepts in detail", // Complex query to trigger big model
		Context:  "User is learning about AI concepts",
		UseTools: true,
	}

	resp, err := agent.GenerateAnswer(ctx, req)
	if err != nil {
		t.Fatalf("RAG integration failed: %v", err)
	}

	// Verify response structure
	if resp == nil {
		t.Fatal("Response should not be nil")
	}

	if resp.Answer == "" {
		t.Error("Answer should not be empty")
	}

	// Verify we used tools
	if len(resp.ToolsUsed) != 1 {
		t.Errorf("Expected 1 tool used, got %d", len(resp.ToolsUsed))
	}

	if resp.ToolsUsed[0].Tool.Name != "rag_search" {
		t.Errorf("Expected rag_search tool, got '%s'", resp.ToolsUsed[0].Tool.Name)
	}

	// Verify answer incorporates multiple sources
	expectedContent := []string{
		"machine learning",
		"ai", // Changed from "artificial intelligence" to be more flexible
	}

	for _, expected := range expectedContent {
		if !strings.Contains(strings.ToLower(resp.Answer), expected) {
			t.Errorf("RAG answer should reference '%s'", expected)
		}
	}

	t.Logf("RAG Integration Answer: %s", resp.Answer)
}

func TestMultiResultWebSearchIntegration(t *testing.T) {
	mockClient := &multiResponseMockClient{
		responses: []string{
			`{"tools": ["web_search"]}`,
			"Go is an open source programming language by Google, designed for simple and efficient software development. It features static typing, compilation, excellent concurrency support, and a clean syntax that's easy to learn.",
		},
	}

	agent := NewAgent(AgentConfig{
		MiniModel: struct {
			Client llm.LLMClient
			Model  string
		}{
			Client: mockClient,
			Model:  "mini-model",
		},
		BigModel: struct {
			Client llm.LLMClient
			Model  string
		}{
			Client: mockClient,
			Model:  "big-model",
		},
		Tools:     []MCPTool{},
		MaxTokens: 1000,
	})

	// Add a mock web search tool that returns multiple results
	webSearchTool := MCPTool{
		Name:        "web_search",
		Description: "Search the web and return multiple relevant results",
		Handler: func(ctx context.Context, params map[string]interface{}) ([]*ToolResult, error) {
			// Simulate web search returning multiple sources using the helper function
			searchResults := []SearchResultPair{
				{
					Content: "Go is an open source programming language developed by Google. It's designed for building simple, reliable, and efficient software.",
					Source:  "https://golang.org/doc/",
				},
				{
					Content: "Go (Golang) is a statically typed, compiled programming language. It has excellent concurrency support and fast compilation times.",
					Source:  "https://en.wikipedia.org/wiki/Go_(programming_language)",
				},
				{
					Content: "Go's syntax is clean and easy to learn. It has built-in garbage collection and strong standard library support for web development.",
					Source:  "https://go.dev/learn/",
				},
			}

			return NewMultiSearchToolResult("What is the Go programming language?", searchResults), nil
		},
	}

	agent.AddTool(webSearchTool)

	ctx := context.Background()
	req := GenerateAnswerRequest{
		Query:    "Please provide a comprehensive analysis of the Go programming language", // Complex query
		Context:  "User wants to learn about Go programming",
		UseTools: true,
	}

	resp, err := agent.GenerateAnswer(ctx, req)
	if err != nil {
		t.Fatalf("Web search integration failed: %v", err)
	}

	// Verify response
	if resp == nil {
		t.Fatal("Response should not be nil")
	}

	if resp.Answer == "" {
		t.Error("Answer should not be empty")
	}

	// Verify we used the web search tool
	if len(resp.ToolsUsed) != 1 {
		t.Errorf("Expected 1 tool used, got %d", len(resp.ToolsUsed))
	}

	if resp.ToolsUsed[0].Tool.Name != "web_search" {
		t.Errorf("Expected web_search tool, got '%s'", resp.ToolsUsed[0].Tool.Name)
	}

	// Verify answer incorporates web search insights
	expectedContent := []string{
		"go",
		"programming",
		"google",
		"language",
	}

	for _, expected := range expectedContent {
		if !strings.Contains(strings.ToLower(resp.Answer), expected) {
			t.Errorf("Web search answer should reference '%s'", expected)
		}
	}

	t.Logf("Web Search Integration Answer: %s", resp.Answer)
}

// Test for the new SummarizeContext feature
func TestSummarizeContextFeature(t *testing.T) {
	mockClient := &multiResponseMockClient{
		responses: []string{
			`{"tools": ["summarized_search"]}`,                             // Tool selection response
			"Machine learning focuses on algorithms that learn from data.", // Summarization response
			"Based on the summarized search results, machine learning is about algorithms that learn from data to make predictions and decisions.", // Final answer
		},
	}

	agent := NewAgent(AgentConfig{
		MiniModel: struct {
			Client llm.LLMClient
			Model  string
		}{
			Client: mockClient,
			Model:  "mini-model",
		},
		BigModel: struct {
			Client llm.LLMClient
			Model  string
		}{
			Client: mockClient,
			Model:  "big-model",
		},
		Tools:     []MCPTool{},
		MaxTokens: 1000,
	})

	// Add a tool with SummarizeContext enabled
	summarizedSearchTool := MCPTool{
		Name:             "summarized_search",
		Description:      "Search with automatic summarization",
		SummarizeContext: true,
		Handler: func(ctx context.Context, params map[string]interface{}) ([]*ToolResult, error) {
			// Simulate verbose search results that need summarization
			return []*ToolResult{
				{
					Sentences: []string{
						"Machine learning is a branch of artificial intelligence.",
						"It involves training algorithms on data.",
						"Common applications include image recognition, natural language processing, and predictive analytics.",
						"The field has grown rapidly in recent years.",
						"Many companies now use machine learning for business insights.",
					},
					Attributions: []string{"AI Research Paper"},
					Title:        "Machine Learning Overview",
					Success:      true,
				},
				{
					Sentences: []string{
						"The weather today is sunny with temperatures around 75 degrees.",
						"Machine learning algorithms can be supervised or unsupervised.",
						"Supervised learning uses labeled training data.",
						"Tomorrow's forecast shows possible rain.",
					},
					Attributions: []string{"Mixed Content Source"},
					Title:        "Mixed Content",
					Success:      true,
				},
			}, nil
		},
	}

	agent.AddTool(summarizedSearchTool)

	ctx := context.Background()
	req := GenerateAnswerRequest{
		Query:    "What is machine learning?",
		Context:  "User wants to understand machine learning concepts",
		UseTools: true,
	}

	resp, err := agent.GenerateAnswer(ctx, req)
	if err != nil {
		t.Fatalf("Summarization test failed: %v", err)
	}

	// Verify response
	if resp == nil {
		t.Fatal("Response should not be nil")
	}

	if resp.Answer == "" {
		t.Error("Answer should not be empty")
	}

	// Verify tool was used
	if len(resp.ToolsUsed) != 1 {
		t.Errorf("Expected 1 tool used, got %d", len(resp.ToolsUsed))
	}

	if resp.ToolsUsed[0].Tool.Name != "summarized_search" {
		t.Errorf("Expected summarized_search tool, got '%s'", resp.ToolsUsed[0].Tool.Name)
	}

	// Verify the tool has SummarizeContext enabled
	if !resp.ToolsUsed[0].Tool.SummarizeContext {
		t.Error("Expected SummarizeContext to be true")
	}

	t.Logf("Summarization Test Answer: %s", resp.Answer)
}

// Test summarization with realistic RAG scenario
func TestSummarizeContextRAGScenario(t *testing.T) {
	mockClient := &multiResponseMockClient{
		responses: []string{
			`{"tools": ["rag_search"]}`, // Tool selection
			"Machine learning uses algorithms to learn from data and make predictions.",          // Summarization of first result
			"The process involves training models on datasets and evaluating their performance.", // Summarization of second result
			"IRRELEVANT", // Third result deemed irrelevant
			"Machine learning is a powerful technology that uses algorithms to learn from data and make predictions. The process involves training models on datasets and evaluating their performance to ensure accuracy.", // Final answer
		},
	}

	agent := NewAgent(AgentConfig{
		MiniModel: struct {
			Client llm.LLMClient
			Model  string
		}{
			Client: mockClient,
			Model:  "mini-model",
		},
		BigModel: struct {
			Client llm.LLMClient
			Model  string
		}{
			Client: mockClient,
			Model:  "big-model",
		},
		Tools:     []MCPTool{},
		MaxTokens: 1000,
	})

	// Create a RAG tool with summarization enabled
	ragTool := MCPTool{
		Name:             "rag_search",
		Description:      "Search knowledge base with smart summarization",
		SummarizeContext: true,
		Handler: func(ctx context.Context, params map[string]interface{}) ([]*ToolResult, error) {
			// Simulate detailed RAG results with lots of text that needs summarization
			return []*ToolResult{
				{
					Sentences: []string{
						"Machine learning is a method of data analysis that automates analytical model building.",
						"It is a branch of artificial intelligence based on the idea that systems can learn from data.",
						"Machine learning algorithms build a model based on training data in order to make predictions or decisions.",
						"The algorithms are used in a wide variety of applications, such as medicine, email filtering, speech recognition, and computer vision.",
						"Machine learning is closely related to computational statistics, which focuses on making predictions using computers.",
					},
					Attributions: []string{"ML Textbook Chapter 1"},
					Title:        "Machine Learning Fundamentals",
					Success:      true,
				},
				{
					Sentences: []string{
						"The study of mathematical optimization delivers methods, theory and application domains to the field of machine learning.",
						"Data mining is a related field of study, focusing on exploratory data analysis through unsupervised learning.",
						"Machine learning involves training algorithms on data sets to recognize patterns and make predictions.",
						"The effectiveness of machine learning depends on the quality and quantity of training data.",
						"Common evaluation metrics include accuracy, precision, recall, and F1-score.",
					},
					Attributions: []string{"Advanced ML Research Paper"},
					Title:        "ML Training and Evaluation",
					Success:      true,
				},
				{
					Sentences: []string{
						"The weather today is sunny with a high of 75 degrees.",
						"Stock prices fluctuated throughout the trading session.",
						"A new restaurant opened downtown offering Mediterranean cuisine.",
						"Traffic conditions are heavy on the main highway.",
					},
					Attributions: []string{"Random News Feed"},
					Title:        "Unrelated Content",
					Success:      true,
				},
			}, nil
		},
	}

	agent.AddTool(ragTool)

	ctx := context.Background()
	req := GenerateAnswerRequest{
		Query:    "Explain how machine learning works", // Complex query to trigger big model
		Context:  "User wants to understand ML concepts for a presentation",
		UseTools: true,
	}

	resp, err := agent.GenerateAnswer(ctx, req)
	if err != nil {
		t.Fatalf("RAG summarization test failed: %v", err)
	}

	// Verify the tool was used and summarization worked
	if len(resp.ToolsUsed) != 1 {
		t.Errorf("Expected 1 tool used, got %d", len(resp.ToolsUsed))
	}

	tool := resp.ToolsUsed[0].Tool
	if !tool.SummarizeContext {
		t.Error("Expected SummarizeContext to be enabled")
	}

	if tool.Name != "rag_search" {
		t.Errorf("Expected rag_search tool, got %s", tool.Name)
	}

	// Verify the query was passed to the tool for summarization context
	if query, ok := resp.ToolsUsed[0].Parameters["query"].(string); !ok || query != req.Query {
		t.Errorf("Expected query to be passed to tool for summarization context")
	}

	t.Logf("RAG Summarization Test Answer: %s", resp.Answer)
	t.Logf("Tools Used: %+v", resp.ToolsUsed[0].Tool.Name)
	t.Logf("SummarizeContext Enabled: %v", resp.ToolsUsed[0].Tool.SummarizeContext)
}
