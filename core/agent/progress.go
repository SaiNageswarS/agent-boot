package agent

import (
	"agent-boot/proto/schema"
	"time"

	"google.golang.org/grpc"
)

// ProgressReporter is an interface for reporting agent execution progress
type ProgressReporter interface {
	// Send sends a progress update
	Send(event *schema.AgentStreamChunk) error
}

// NoOpProgressReporter implements ProgressReporter with no-op operations
type NoOpProgressReporter struct{}

// Send does nothing
func (r *NoOpProgressReporter) Send(event *schema.AgentStreamChunk) error {
	// No-op
	return nil
}

// GrpcProgressReporter implements ProgressReporter for gRPC streaming
type GrpcProgressReporter struct {
	stream grpc.ServerStreamingServer[schema.AgentStreamChunk]
}

func (r *GrpcProgressReporter) Send(event *schema.AgentStreamChunk) error {
	return r.stream.Send(event)
}

// Helper functions for creating progress events
func NewProgressUpdate(stage schema.Stage, message string, current int32) *schema.AgentStreamChunk {
	return &schema.AgentStreamChunk{
		ChunkType: &schema.AgentStreamChunk_ProgressUpdateChunk{
			ProgressUpdateChunk: &schema.ProgressUpdateChunk{
				Stage:          stage,
				Timestamp:      time.Now().UnixMilli(),
				Message:        message,
				CurrentStep:    current,
				EstimatedSteps: 3,
			},
		},
	}
}

// NewToolSelectionResult creates a ToolSelectionResult chunk
func NewToolSelectionResult(selectedTool *schema.SelectedTool) *schema.AgentStreamChunk {
	return &schema.AgentStreamChunk{
		ChunkType: &schema.AgentStreamChunk_SelectedTool{
			SelectedTool: selectedTool,
		},
	}
}

// NewToolExecutionResult creates a ToolExecutionResultChunk chunk
func NewToolExecutionResult(toolName string, result *schema.ToolExecutionResultChunk) *schema.AgentStreamChunk {
	result.ToolName = toolName

	return &schema.AgentStreamChunk{
		ChunkType: &schema.AgentStreamChunk_ToolExecutionResultChunk{
			ToolExecutionResultChunk: result,
		},
	}
}

// NewAnswerChunk creates an AnswerChunk
func NewAnswerChunk(answer string, toolsUsed []string, tokens int32, prompt, model string, meta map[string]string, isFinal bool) *schema.AgentStreamChunk {
	return &schema.AgentStreamChunk{
		ChunkType: &schema.AgentStreamChunk_Answer{
			Answer: &schema.AnswerChunk{
				Answer:         answer,
				ToolsUsed:      toolsUsed,
				TokenUsed:      tokens,
				PromptUsed:     prompt,
				ModelUsed:      model,
				Metadata:       meta,
				IsFinal:        isFinal,
				ProcessingTime: time.Now().UnixMilli(),
			},
		},
	}
}

// NewStreamComplete creates a StreamComplete chunk
func NewStreamComplete(status string) *schema.AgentStreamChunk {
	return &schema.AgentStreamChunk{
		ChunkType: &schema.AgentStreamChunk_Complete{
			Complete: &schema.StreamComplete{
				FinalStatus: status,
			},
		},
	}
}

// NewStreamError creates a StreamError chunk
func NewStreamError(message, code string) *schema.AgentStreamChunk {
	return &schema.AgentStreamChunk{
		ChunkType: &schema.AgentStreamChunk_Error{
			Error: &schema.StreamError{
				ErrorMessage: message,
				ErrorCode:    code,
			},
		},
	}
}
