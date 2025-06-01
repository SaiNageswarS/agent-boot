package workers

type Chunk struct {
	Header    string `json:"header"`
	Page      int    `json:"page"`
	Objective string `json:"objective"`
	Passage   string `json:"passage"`
	SourcePdf string `json:"sourcePdf"` // URL or path to the source PDF file
}

type IndexerWorkflowInput struct {
	PdfUrl string `json:"pdfUrl"`
	Tenant string `json:"tenant"` // Tenant ID or name
}
