package workers

type Chunk struct {
	ChunkID      string   `json:"chunkId"`
	SectionPath  []string `json:"sectionPath"`
	SectionIndex int      `json:"sectionIndex"` // Index of the section in the path
	CreatedAt    string   `json:"createdAt"`    // ISO 8601 format
	Embedding    string   `json:"embedding"`    // e.g., "text-embedding-3-small"
	PHIRemoved   bool     `json:"phiRemoved"`   // true if PHI is removed, false otherwise
	SourceURI    string   `json:"sourceUri"`    // e.g., "file://path/to/file.pdf"
	Body         string   `json:"body"`         // The actual content of the chunk
}

type IndexerWorkflowState struct {
	PdfFile            string `json:"pdfFile"`
	Tenant             string `json:"tenant"`
	Markdown           string `json:"markdown"` // For future use, if needed
	MdSectionChunksUrl string `json:"mdSectionChunksUrl"`
}
