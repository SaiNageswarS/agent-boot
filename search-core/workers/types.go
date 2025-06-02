package workers

type Chunk struct {
	ID        string      `json:"id"`
	Text      string      `json:"text"`
	Page      int         `json:"page"`
	Vectors   [][]float32 // [general, medical, objective]
	Summary   string      `json:"summary"`
	SourcePDF string      `json:"sourcePdf"`
}

type IndexerWorkflowState struct {
	PdfFile  string `json:"pdfFile"`
	Tenant   string `json:"tenant"`
	Markdown string `json:"markdown"` // For future use, if needed
}
