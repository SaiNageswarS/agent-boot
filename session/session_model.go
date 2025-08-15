package session

import "github.com/SaiNageswarS/agent-boot/llm"

type SessionModel struct {
	ID       string        `bson:"_id"`
	Messages []llm.Message `bson:"messages"`
}

func (m SessionModel) Id() string {
	return m.ID
}

func (m SessionModel) CollectionName() string {
	return "sessions"
}
