package prompts

import (
	"bytes"
	"embed"
	"html/template"
	"strings"
)

//go:embed templates/*
var templatesFS embed.FS

func loadPrompt(templatePath string, data interface{}) (string, error) {
	tmpl, err := template.ParseFS(templatesFS, templatePath)
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	err = tmpl.Execute(&buf, data)
	if err != nil {
		return "", err
	}

	return buf.String(), nil
}

func extractSection(response, sectionHeader string) []string {
	lines := strings.Split(response, "\n")
	var summaryLines []string
	inSummary := false
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, sectionHeader) {
			inSummary = true
			continue
		}
		if inSummary {
			if line == "" || strings.HasPrefix(line, "THOUGHTS:") {
				break // End of SUMMARY block
			}
			summaryLines = append(summaryLines, line)
		}
	}

	return summaryLines
}
