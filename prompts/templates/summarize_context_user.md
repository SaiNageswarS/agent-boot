**User Question:** {{.Query}}

{{if .ToolInputs}}**Tool Inputs:**
{{.ToolInputs}}

{{end}}**Content to Summarize:**
{{.Content}}

Please summarize the above content with respect to the user's question{{if .ToolInputs}} and tool inputs{{end}}. If the content is irrelevant to the question{{if .ToolInputs}} and tool inputs{{end}}, respond with "# IRRELEVANT".
