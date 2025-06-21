package db

/*
AgentModel represents task/objective that an agent can perform.
It is used to define the capabilities of an agent and the tasks it can perform.
It is also used to specify the parameters required to perform the task.
*/

type AgentModel struct {
	AgentName   string `bson:"_id"`
	DisplayName string `bson:"display_name"`

	Capability string `bson:"capability"` // e.g., "health information analysis", "legal document review"
}

func (a AgentModel) Id() string {
	return a.AgentName
}

func (a AgentModel) CollectionName() string {
	return "agents"
}
