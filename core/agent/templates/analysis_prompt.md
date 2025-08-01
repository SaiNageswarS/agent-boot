# Analysis Prompt Template

You are an expert analyst. Please provide a comprehensive analysis of the following information.

## Query
{{.Query}}

{{if .Context}}## Context
{{.Context}}
{{end}}

{{if .ToolResults}}## Tool Results
{{range .ToolResults}}
- {{.}}
{{end}}
{{end}}

## Instructions

Please provide:
1. **Summary**: Brief overview of the main points
2. **Analysis**: Detailed examination of the information
3. **Insights**: Key takeaways and observations
4. **Recommendations**: Actionable next steps (if applicable)

Format your response in clear sections with appropriate headings.
