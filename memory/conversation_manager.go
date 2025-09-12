package memory

import (
	"context"

	"github.com/SaiNageswarS/agent-boot/llm"
	"github.com/SaiNageswarS/go-api-boot/logger"
	"github.com/SaiNageswarS/go-api-boot/odm"
	"github.com/SaiNageswarS/go-collection-boot/async"
	"go.uber.org/zap"
)

// ConversationManager handles conversation-related operations
type ConversationManager struct {
	collection odm.OdmCollectionInterface[Conversation]
	maxMsgs    int
}

// NewConversationManager creates a new conversation manager
func NewConversationManager(collection odm.OdmCollectionInterface[Conversation], maxMsgs int) *ConversationManager {
	return &ConversationManager{
		collection: collection,
		maxMsgs:    maxMsgs,
	}
}

// LoadSession loads previous conversation messages for a session
func (cm *ConversationManager) LoadSession(ctx context.Context, sessionID string) *Conversation {
	if cm.collection == nil {
		return &Conversation{}
	}

	session, err := async.Await(cm.collection.FindOneByID(ctx, sessionID))
	if err != nil {
		logger.Error("Failed to find session", zap.Error(err))
		return &Conversation{} // Return empty slice instead of error to allow conversation to continue
	}

	return session
}

// SaveSession saves the conversation messages for a session
func (cm *ConversationManager) SaveSession(ctx context.Context, conversation *Conversation) error {
	if cm.collection == nil {
		return nil
	}

	// Trim messages to respect max session limit
	conversation.Messages = cm.trimForSession(conversation.Messages)

	_, err := async.Await(cm.collection.Save(ctx, *conversation))
	if err != nil {
		logger.Error("Failed to save session", zap.Error(err))
		return err
	}

	return nil
}

// trimForSession keeps the last maxMsgs "user" messages and any number of
// "assistant" (and optional "tool") messages that follow them.
// If there are fewer than maxMsgs user messages total, it returns msgs unchanged.
func (cm *ConversationManager) trimForSession(msgs []llm.Message) []llm.Message {
	if cm.maxMsgs <= 0 || len(msgs) == 0 {
		return []llm.Message{}
	}

	// Walk backward and find the boundary index: the position right after the
	// (maxMsgs+1)-th user from the end. Everything after boundary is kept.
	usersSeen := 0
	start := 0 // default: keep all if we don't exceed maxMsgs users
	for i := len(msgs) - 1; i >= 0; i-- {
		if msgs[i].Role == "user" && !msgs[i].IsToolResult {
			usersSeen++
			if usersSeen == cm.maxMsgs {
				start = i
				break
			}
		}
	}

	return msgs[start:]
}

// GetMaxMessages returns the maximum number of messages allowed in a session
func (cm *ConversationManager) GetMaxMessages() int {
	return cm.maxMsgs
}

// SetMaxMessages sets the maximum number of messages allowed in a session
func (cm *ConversationManager) SetMaxMessages(max int) {
	cm.maxMsgs = max
}
