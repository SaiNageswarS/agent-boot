# Template Directory

This directory contains embedded Markdown templates used by the Agent Boot system. These templates are compiled into the Go binary using Go's `embed` directive.

## Available Templates

### Tool Selection Templates

#### `tool_selection_system.md`
System prompt for the AI tool selection expert. Defines the role, available tools, and response format.

**Variables:**
- `.ToolDescriptions` - Array of tool descriptions
- `.MaxTools` - Maximum number of tools to select

#### `tool_selection_user.md`
User prompt for tool selection requests.

**Variables:**
- `.Query` - User's query
- `.Context` - Additional context (optional)

### Answer Generation Templates

#### `analysis_prompt.md`
Template for comprehensive analysis tasks. Provides structured format for analysis with sections for summary, analysis, insights, and recommendations.

**Variables:**
- `.Query` - The analysis query
- `.Context` - Analysis context (optional)
- `.ToolResults` - Array of tool results (optional)

#### `default_answer.md`
General-purpose template for answer generation with query, context, and tool results.

**Variables:**
- `.Query` - User's question
- `.Context` - Additional context (optional)
- `.ToolResults` - Array of tool results (optional)

## Template Syntax

Templates use Go's `text/template` syntax:

### Variables
```
{{.VariableName}}
```

### Conditional Content
```
{{if .Context}}
Content shown only if Context is not empty
{{end}}
```

### Loops
```
{{range .ToolResults}}
- {{.}}
{{end}}
```

## Adding New Templates

1. Create a new `.md` file in this directory
2. Use Go template syntax for variables
3. Update `template_manager.go` to include the new template file
4. Add appropriate data structures if needed

## Example Template Structure

```markdown
# Template Title

## Section 1
{{.Variable1}}

{{if .OptionalVariable}}
## Optional Section
{{.OptionalVariable}}
{{end}}

{{if .ArrayVariable}}
## List Section
{{range .ArrayVariable}}
- {{.}}
{{end}}
{{end}}

## Instructions
Your instructions here...
```

## Best Practices

1. **Clear Structure**: Use Markdown headers to organize content
2. **Conditional Sections**: Use `{{if}}` for optional content
3. **Variable Names**: Use descriptive names that match the data structure
4. **Documentation**: Include comments about expected variables
5. **Formatting**: Maintain consistent Markdown formatting

## Template Data Types

The system uses the following data structures:

- `ToolSelectionPromptData` - For tool selection templates
- `PromptData` - Generic data structure for most templates

Refer to the Go source files for complete data structure definitions.
