package workers

type Chunk struct {
	ID        string      `json:"id"`
	Text      string      `json:"text"`
	Page      int         `json:"page"`
	Vectors   [][]float32 // [general, medical, objective]
	Summary   string      `json:"summary"`
	SourcePDF string      `json:"sourcePdf"`
}

type IndexerWorkflowInput struct {
	PdfFile string `json:"pdfFile"`
	Tenant  string `json:"tenant"`
}
