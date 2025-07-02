package db

// A single exchange between the user and the agent.
// It contains the user input, search queries, search result chunk IDs, agent's answer,
// and the model used to generate the answer.
// This is used to track the conversation history and the agent's responses.
type TurnModel struct {
	UserInput            string   `bson:"userInput"`
	SearchQueries        []string `bson:"searchQueries"`
	SearchResultChunkIds []string `bson:"searchResultChunkIds"`
	AgentAnswer          string   `bson:"agentAnswer"`
	Model                string   `bson:"model"`
}

type SessionModel struct {
	ID        string      `bson:"_id"`
	UserId    string      `bson:"userId"`
	Turns     []TurnModel `bson:"turns"`
	CreatedOn int64       `bson:"createdOn"`
}

func NewSessionModel(userId, sessionId string) *SessionModel {
	return &SessionModel{
		ID:     sessionId,
		UserId: userId,
		Turns:  []TurnModel{},
	}
}

func (m SessionModel) Id() string {
	return m.ID
}

func (m SessionModel) CollectionName() string {
	return "sessions"
}
