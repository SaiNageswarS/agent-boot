# Unified Inference Query

**Query:** {{.Query}}

{{if .Context}}**Context:** {{.Context}}{{end}}

{{if .CurrentTurn}}**Turn:** {{.CurrentTurn}}/{{.MaxTurns}}{{end}}

{{if .PreviousToolResults}}**Previous Tool Results:**
{{.PreviousToolResults}}{{end}}

Analyze this query and respond appropriately using either tools or a direct answer.
