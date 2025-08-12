package agentboot

import (
	"time"

	"github.com/SaiNageswarS/agent-boot/schema"
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
	Stream grpc.ServerStreamingServer[schema.AgentStreamChunk]
}

func (r *GrpcProgressReporter) Send(event *schema.AgentStreamChunk) error {
	return r.Stream.Send(event)
}

// Helper functions for creating progress events
func NewProgressUpdate(stage schema.Stage, message string) *schema.AgentStreamChunk {
	return &schema.AgentStreamChunk{
		ChunkType: &schema.AgentStreamChunk_ProgressUpdateChunk{
			ProgressUpdateChunk: &schema.ProgressUpdateChunk{
				Stage:          stage,
				Timestamp:      time.Now().UnixMilli(),
				Message:        message,
				EstimatedSteps: 3,
			},
		},
	}
}

// NewToolExecutionResult creates a ToolExecutionResultChunk chunk
func NewToolExecutionResult(toolName string, result *schema.ToolResultChunk) *schema.AgentStreamChunk {
	result.ToolName = toolName

	return &schema.AgentStreamChunk{
		ChunkType: &schema.AgentStreamChunk_ToolResultChunk{
			ToolResultChunk: result,
		},
	}
}

// NewAnswerChunk creates an AnswerChunk
func NewAnswerChunk(answerChunk *schema.AnswerChunk) *schema.AgentStreamChunk {
	return &schema.AgentStreamChunk{
		ChunkType: &schema.AgentStreamChunk_Answer{
			Answer: answerChunk,
		},
	}
}

// NewStreamComplete creates a StreamComplete chunk
func NewStreamComplete(finalResponse *schema.StreamComplete) *schema.AgentStreamChunk {
	return &schema.AgentStreamChunk{
		ChunkType: &schema.AgentStreamChunk_Complete{
			Complete: finalResponse,
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
