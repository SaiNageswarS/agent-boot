package agent

import "time"

// ProgressEventType represents the type of progress event
type ProgressEventType string

const (
	// ProgressEventToolSelection indicates tool selection is in progress
	ProgressEventToolSelection ProgressEventType = "tool_selection"

	// ProgressEventToolExecution indicates a tool is being executed
	ProgressEventToolExecution ProgressEventType = "tool_execution"

	// ProgressEventToolResult indicates a tool has completed execution
	ProgressEventToolResult ProgressEventType = "tool_result"

	// ProgressEventAnswerGeneration indicates answer generation is in progress
	ProgressEventAnswerGeneration ProgressEventType = "answer_generation"

	// ProgressEventAnswerChunk indicates a chunk of the answer has been generated
	ProgressEventAnswerChunk ProgressEventType = "answer_chunk"

	// ProgressEventCompleted indicates the process has completed
	ProgressEventCompleted ProgressEventType = "completed"

	// ProgressEventError indicates an error occurred
	ProgressEventError ProgressEventType = "error"
)

// ProgressEvent represents a progress update during agent execution
type ProgressEvent struct {
	// Type of the progress event
	Type ProgressEventType `json:"type"`

	// Timestamp when the event occurred
	Timestamp time.Time `json:"timestamp"`

	// Stage description
	Stage string `json:"stage"`

	// Message describing the current progress
	Message string `json:"message"`

	// Data contains event-specific data
	Data interface{} `json:"data,omitempty"`

	// Error contains error information if Type is ProgressEventError
	Error string `json:"error,omitempty"`
}

// ToolSelectionProgress represents progress during tool selection
type ToolSelectionProgress struct {
	Query      string `json:"query"`
	ToolsCount int    `json:"tools_count"`
	MaxTools   int    `json:"max_tools"`
	Status     string `json:"status"` // "starting", "processing", "completed"
}

// ToolExecutionProgress represents progress during tool execution
type ToolExecutionProgress struct {
	ToolName   string                 `json:"tool_name"`
	Parameters map[string]interface{} `json:"parameters"`
	Status     string                 `json:"status"` // "starting", "executing", "completed", "failed"
}

// ToolResultProgress represents the result of tool execution
type ToolResultProgress struct {
	ToolName   string             `json:"tool_name"`
	Results    []*ToolResultChunk `json:"results"`
	Success    bool               `json:"success"`
	Duration   time.Duration      `json:"duration"`
	Summarized bool               `json:"summarized,omitempty"`
}

// AnswerGenerationProgress represents progress during answer generation
type AnswerGenerationProgress struct {
	ModelUsed    string `json:"model_used"`
	PromptLength int    `json:"prompt_length"`
	Status       string `json:"status"` // "starting", "generating", "completed"
}

// AnswerChunkProgress represents a chunk of the generated answer
type AnswerChunkProgress struct {
	Chunk       string `json:"chunk"`
	TotalLength int    `json:"total_length"`
	IsComplete  bool   `json:"is_complete"`
}

// CompletionProgress represents the final completion status
type CompletionProgress struct {
	TotalDuration time.Duration `json:"total_duration"`
	ToolsUsed     int           `json:"tools_used"`
	ModelUsed     string        `json:"model_used"`
	AnswerLength  int           `json:"answer_length"`
	Success       bool          `json:"success"`
}

// ProgressReporter is an interface for reporting agent execution progress
type ProgressReporter interface {
	// ReportProgress sends a progress update
	ReportProgress(event ProgressEvent)
}

// ChannelProgressReporter implements ProgressReporter using a channel
type ChannelProgressReporter struct {
	progressChan chan<- ProgressEvent
}

// NewChannelProgressReporter creates a new channel-based progress reporter
func NewChannelProgressReporter(progressChan chan<- ProgressEvent) *ChannelProgressReporter {
	return &ChannelProgressReporter{
		progressChan: progressChan,
	}
}

// ReportProgress sends progress to the channel
func (r *ChannelProgressReporter) ReportProgress(event ProgressEvent) {
	select {
	case r.progressChan <- event:
		// Progress sent successfully
	default:
		// Channel is full or closed, skip this progress update
	}
}

// FunctionProgressReporter implements ProgressReporter using a callback function
type FunctionProgressReporter struct {
	callback func(ProgressEvent)
}

// NewFunctionProgressReporter creates a new function-based progress reporter
func NewFunctionProgressReporter(callback func(ProgressEvent)) *FunctionProgressReporter {
	return &FunctionProgressReporter{
		callback: callback,
	}
}

// ReportProgress calls the callback function
func (r *FunctionProgressReporter) ReportProgress(event ProgressEvent) {
	if r.callback != nil {
		r.callback(event)
	}
}

// NoOpProgressReporter implements ProgressReporter with no-op operations
type NoOpProgressReporter struct{}

// ReportProgress does nothing
func (r *NoOpProgressReporter) ReportProgress(event ProgressEvent) {
	// No-op
}

// Helper functions for creating progress events

func NewToolSelectionEvent(stage, message string, data *ToolSelectionProgress) ProgressEvent {
	return ProgressEvent{
		Type:      ProgressEventToolSelection,
		Timestamp: time.Now(),
		Stage:     stage,
		Message:   message,
		Data:      data,
	}
}

func NewToolExecutionEvent(stage, message string, data *ToolExecutionProgress) ProgressEvent {
	return ProgressEvent{
		Type:      ProgressEventToolExecution,
		Timestamp: time.Now(),
		Stage:     stage,
		Message:   message,
		Data:      data,
	}
}

func NewToolResultEvent(stage, message string, data *ToolResultProgress) ProgressEvent {
	return ProgressEvent{
		Type:      ProgressEventToolResult,
		Timestamp: time.Now(),
		Stage:     stage,
		Message:   message,
		Data:      data,
	}
}

func NewAnswerGenerationEvent(stage, message string, data *AnswerGenerationProgress) ProgressEvent {
	return ProgressEvent{
		Type:      ProgressEventAnswerGeneration,
		Timestamp: time.Now(),
		Stage:     stage,
		Message:   message,
		Data:      data,
	}
}

func NewAnswerChunkEvent(stage, message string, data *AnswerChunkProgress) ProgressEvent {
	return ProgressEvent{
		Type:      ProgressEventAnswerChunk,
		Timestamp: time.Now(),
		Stage:     stage,
		Message:   message,
		Data:      data,
	}
}

func NewCompletionEvent(stage, message string, data *CompletionProgress) ProgressEvent {
	return ProgressEvent{
		Type:      ProgressEventCompleted,
		Timestamp: time.Now(),
		Stage:     stage,
		Message:   message,
		Data:      data,
	}
}

func NewErrorEvent(stage, message, errorMsg string) ProgressEvent {
	return ProgressEvent{
		Type:      ProgressEventError,
		Timestamp: time.Now(),
		Stage:     stage,
		Message:   message,
		Error:     errorMsg,
	}
}
