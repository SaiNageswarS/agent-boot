package workflows

type ChunkMarkdownWorkflowInput struct {
	MarkdownFile string `json:"markdownFile"` // Path to the markdown file
	Tenant       string `json:"tenant"`
	SourceUri    string `json:"sourceUri"` // URI of the source file
}

type InitTenantWorkflowInput struct {
	Tenant string `json:"tenant"`
}

type PdfHandlerWorkflowInput struct {
	PdfFile string `json:"pdfFile"`
	Tenant  string `json:"tenant"`
}

type EmbedChunksWorkflowInput struct {
	Tenant    string `json:"tenant"`
	SourceUri string `json:"sourceUri"` // URI of the source file
}
