# Default Answer Generation Prompt

## Query
{{.Query}}

{{if .Context}}## Additional Context
{{.Context}}
{{end}}

{{if .ToolResults}}## Information from Tools
{{range .ToolResults}}
{{.}}

{{end}}
{{end}}

Please provide a comprehensive answer based on the above information. Be accurate, helpful, and well-structured in your response.
