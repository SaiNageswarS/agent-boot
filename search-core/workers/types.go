package workers

type IndexerWorkflowState struct {
	PdfFile            string   `json:"pdfFile"`
	Tenant             string   `json:"tenant"`
	Enhancement        string   `json:"enhancement"` // e.g., "medical_entities"
	MarkdownFile       string   `json:"markdownFile"`
	MdSectionChunkUrls []string `json:"mdSectionChunksUrls"`
	WindowChunkUrls    []string `json:"windowChunkUrls"` // URL for window chunks
}
