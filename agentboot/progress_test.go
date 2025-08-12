package agentboot

import (
	"testing"
	"time"

	"github.com/SaiNageswarS/agent-boot/schema"
	"github.com/stretchr/testify/assert"
)

func TestNoOpProgressReporter(t *testing.T) {
	reporter := &NoOpProgressReporter{}

	// Test that Send does nothing and returns no error
	chunk := &schema.AgentStreamChunk{}
	err := reporter.Send(chunk)

	assert.NoError(t, err)
}

func TestNoOpProgressReporterMultipleCalls(t *testing.T) {
	reporter := &NoOpProgressReporter{}

	// Test multiple calls
	for i := 0; i < 10; i++ {
		chunk := &schema.AgentStreamChunk{}
		err := reporter.Send(chunk)
		assert.NoError(t, err)
	}
}

func TestNewProgressUpdate(t *testing.T) {
	stage := schema.Stage_tool_execution_starting
	message := "Starting tool execution"

	chunk := NewProgressUpdate(stage, message)

	assert.NotNil(t, chunk)
	assert.NotNil(t, chunk.ChunkType)

	progressChunk := chunk.GetProgressUpdateChunk()
	assert.NotNil(t, progressChunk)
	assert.Equal(t, stage, progressChunk.Stage)
	assert.Equal(t, message, progressChunk.Message)
	assert.Equal(t, int32(3), progressChunk.EstimatedSteps)
	assert.Greater(t, progressChunk.Timestamp, int64(0))

	// Verify timestamp is recent (within last minute)
	now := time.Now().UnixMilli()
	assert.True(t, progressChunk.Timestamp <= now)
	assert.True(t, progressChunk.Timestamp > now-60000) // Within last minute
}

func TestNewToolExecutionResult(t *testing.T) {
	toolName := "calculator"
	result := &schema.ToolResultChunk{
		Sentences:   []string{"2 + 2 = 4"},
		Attribution: "Calculator tool",
		Title:       "Math Result",
		Metadata:    map[string]string{"operation": "addition"},
	}

	chunk := NewToolExecutionResult(toolName, result)

	assert.NotNil(t, chunk)
	assert.NotNil(t, chunk.ChunkType)

	toolChunk := chunk.GetToolResultChunk()
	assert.NotNil(t, toolChunk)
	assert.Equal(t, toolName, toolChunk.ToolName) // Should be set by the function
	assert.Equal(t, result.Sentences, toolChunk.Sentences)
	assert.Equal(t, result.Attribution, toolChunk.Attribution)
	assert.Equal(t, result.Title, toolChunk.Title)
	assert.Equal(t, result.Metadata, toolChunk.Metadata)
}

func TestNewAnswerChunk(t *testing.T) {
	answerChunk := &schema.AnswerChunk{
		Content: "This is the answer to your question.",
	}

	chunk := NewAnswerChunk(answerChunk)

	assert.NotNil(t, chunk)
	assert.NotNil(t, chunk.ChunkType)

	retrievedAnswerChunk := chunk.GetAnswer()
	assert.NotNil(t, retrievedAnswerChunk)
	assert.Equal(t, answerChunk.Content, retrievedAnswerChunk.Content)
}

func TestNewStreamComplete(t *testing.T) {
	finalResponse := &schema.StreamComplete{
		Answer:         "Final answer",
		TokenUsed:      150,
		ProcessingTime: 2500,
		Metadata:       map[string]string{"model": "test-model"},
		ToolsUsed:      []string{"calculator", "search"},
	}

	chunk := NewStreamComplete(finalResponse)

	assert.NotNil(t, chunk)
	assert.NotNil(t, chunk.ChunkType)

	completeChunk := chunk.GetComplete()
	assert.NotNil(t, completeChunk)
	assert.Equal(t, finalResponse.Answer, completeChunk.Answer)
	assert.Equal(t, finalResponse.TokenUsed, completeChunk.TokenUsed)
	assert.Equal(t, finalResponse.ProcessingTime, completeChunk.ProcessingTime)
	assert.Equal(t, finalResponse.Metadata, completeChunk.Metadata)
	assert.Equal(t, finalResponse.ToolsUsed, completeChunk.ToolsUsed)
}

func TestNewStreamError(t *testing.T) {
	message := "An error occurred"
	code := "TOOL_EXECUTION_FAILED"

	chunk := NewStreamError(message, code)

	assert.NotNil(t, chunk)
	assert.NotNil(t, chunk.ChunkType)

	errorChunk := chunk.GetError()
	assert.NotNil(t, errorChunk)
	assert.Equal(t, message, errorChunk.ErrorMessage)
	assert.Equal(t, code, errorChunk.ErrorCode)
}

func TestProgressUpdateWithDifferentStages(t *testing.T) {
	stages := []schema.Stage{
		schema.Stage_tool_execution_starting,
		schema.Stage_tool_execution_failed,
		schema.Stage_tool_execution_completed,
		schema.Stage_answer_generation_starting,
		schema.Stage_answer_generation_failed,
		schema.Stage_answer_generation_completed,
	}

	for _, stage := range stages {
		message := "Test message for stage"
		chunk := NewProgressUpdate(stage, message)

		assert.NotNil(t, chunk)
		progressChunk := chunk.GetProgressUpdateChunk()
		assert.NotNil(t, progressChunk)
		assert.Equal(t, stage, progressChunk.Stage)
		assert.Equal(t, message, progressChunk.Message)
	}
}

func TestToolExecutionResultWithEmptyResult(t *testing.T) {
	toolName := "empty-tool"
	result := &schema.ToolResultChunk{}

	chunk := NewToolExecutionResult(toolName, result)

	assert.NotNil(t, chunk)
	toolChunk := chunk.GetToolResultChunk()
	assert.NotNil(t, toolChunk)
	assert.Equal(t, toolName, toolChunk.ToolName)
	assert.Empty(t, toolChunk.Sentences)
	assert.Empty(t, toolChunk.Attribution)
	assert.Empty(t, toolChunk.Title)
	assert.Empty(t, toolChunk.Metadata)
}

func TestAnswerChunkWithEmptyContent(t *testing.T) {
	answerChunk := &schema.AnswerChunk{
		Content: "",
	}

	chunk := NewAnswerChunk(answerChunk)

	assert.NotNil(t, chunk)
	retrievedAnswerChunk := chunk.GetAnswer()
	assert.NotNil(t, retrievedAnswerChunk)
	assert.Empty(t, retrievedAnswerChunk.Content)
}

func TestStreamCompleteWithEmptyFields(t *testing.T) {
	finalResponse := &schema.StreamComplete{
		Answer:         "",
		TokenUsed:      0,
		ProcessingTime: 0,
		Metadata:       map[string]string{},
		ToolsUsed:      []string{},
	}

	chunk := NewStreamComplete(finalResponse)

	assert.NotNil(t, chunk)
	completeChunk := chunk.GetComplete()
	assert.NotNil(t, completeChunk)
	assert.Empty(t, completeChunk.Answer)
	assert.Equal(t, int32(0), completeChunk.TokenUsed)
	assert.Equal(t, int64(0), completeChunk.ProcessingTime)
	assert.Empty(t, completeChunk.Metadata)
	assert.Empty(t, completeChunk.ToolsUsed)
}

func TestStreamErrorWithEmptyFields(t *testing.T) {
	chunk := NewStreamError("", "")

	assert.NotNil(t, chunk)
	errorChunk := chunk.GetError()
	assert.NotNil(t, errorChunk)
	assert.Empty(t, errorChunk.ErrorMessage)
	assert.Empty(t, errorChunk.ErrorCode)
}

func TestChunkTypeDiscrimination(t *testing.T) {
	// Test that each chunk type is properly discriminated
	progressChunk := NewProgressUpdate(schema.Stage_tool_execution_starting, "test")
	assert.NotNil(t, progressChunk.GetProgressUpdateChunk())
	assert.Nil(t, progressChunk.GetToolResultChunk())
	assert.Nil(t, progressChunk.GetAnswer())
	assert.Nil(t, progressChunk.GetComplete())
	assert.Nil(t, progressChunk.GetError())

	toolChunk := NewToolExecutionResult("test", &schema.ToolResultChunk{})
	assert.Nil(t, toolChunk.GetProgressUpdateChunk())
	assert.NotNil(t, toolChunk.GetToolResultChunk())
	assert.Nil(t, toolChunk.GetAnswer())
	assert.Nil(t, toolChunk.GetComplete())
	assert.Nil(t, toolChunk.GetError())

	answerChunk := NewAnswerChunk(&schema.AnswerChunk{})
	assert.Nil(t, answerChunk.GetProgressUpdateChunk())
	assert.Nil(t, answerChunk.GetToolResultChunk())
	assert.NotNil(t, answerChunk.GetAnswer())
	assert.Nil(t, answerChunk.GetComplete())
	assert.Nil(t, answerChunk.GetError())

	completeChunk := NewStreamComplete(&schema.StreamComplete{})
	assert.Nil(t, completeChunk.GetProgressUpdateChunk())
	assert.Nil(t, completeChunk.GetToolResultChunk())
	assert.Nil(t, completeChunk.GetAnswer())
	assert.NotNil(t, completeChunk.GetComplete())
	assert.Nil(t, completeChunk.GetError())

	errorChunk := NewStreamError("test", "TEST_ERROR")
	assert.Nil(t, errorChunk.GetProgressUpdateChunk())
	assert.Nil(t, errorChunk.GetToolResultChunk())
	assert.Nil(t, errorChunk.GetAnswer())
	assert.Nil(t, errorChunk.GetComplete())
	assert.NotNil(t, errorChunk.GetError())
}

// Benchmark tests
func BenchmarkNoOpProgressReporter(b *testing.B) {
	reporter := &NoOpProgressReporter{}
	chunk := &schema.AgentStreamChunk{}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := reporter.Send(chunk)
		_ = err
	}
}

func BenchmarkNewProgressUpdate(b *testing.B) {
	stage := schema.Stage_tool_execution_starting
	message := "Test message"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		chunk := NewProgressUpdate(stage, message)
		_ = chunk
	}
}

func BenchmarkNewToolExecutionResult(b *testing.B) {
	toolName := "test-tool"
	result := &schema.ToolResultChunk{
		Sentences: []string{"test sentence"},
		Title:     "test title",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		chunk := NewToolExecutionResult(toolName, result)
		_ = chunk
	}
}

func BenchmarkNewAnswerChunk(b *testing.B) {
	answerChunk := &schema.AnswerChunk{
		Content: "Test answer content",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		chunk := NewAnswerChunk(answerChunk)
		_ = chunk
	}
}

func BenchmarkNewStreamComplete(b *testing.B) {
	finalResponse := &schema.StreamComplete{
		Answer:         "Test answer",
		TokenUsed:      100,
		ProcessingTime: 1000,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		chunk := NewStreamComplete(finalResponse)
		_ = chunk
	}
}

func BenchmarkNewStreamError(b *testing.B) {
	message := "Test error message"
	code := "TEST_ERROR"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		chunk := NewStreamError(message, code)
		_ = chunk
	}
}
