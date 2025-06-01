package workers

import (
	"context"
	"encoding/json"
	"errors"
	"path/filepath"

	"github.com/SaiNageswarS/agent-boot/search-core/appconfig"
	"github.com/SaiNageswarS/go-api-boot/cloud"
	"github.com/SaiNageswarS/go-api-boot/llm"
)

type IndexerActivities struct {
	ccfg    *appconfig.AppConfig
	cloud   cloud.Cloud
	chunker *Chunker
}

func ProvideIndexerActivities(ccfg *appconfig.AppConfig, cloud cloud.Cloud, llmClient *llm.AnthropicClient) *IndexerActivities {
	return &IndexerActivities{
		ccfg:    ccfg,
		cloud:   cloud,
		chunker: ProvideChunker(),
	}
}

func (s *IndexerActivities) ChunkPDF(ctx context.Context, tenant, pdfUrl string) (string, error) {
	// Download the PDF file to a temporary location
	pdfPath, err := s.cloud.DownloadFile(ctx, s.ccfg.SearchIndexBucket, tenant+"/"+pdfUrl)
	if err != nil {
		return "", errors.New("failed to download PDF file: " + err.Error())
	}

	// Chunk the PDF file
	chunks, err := s.chunker.ChunkPDF(ctx, pdfPath)
	if err != nil {
		return "", errors.New("failed to chunk PDF file: " + err.Error())
	}

	for _, chunk := range chunks {
		chunk.SourcePdf = pdfUrl
	}

	// Convert chunks to JSON
	chunksJson, err := json.Marshal(chunks)
	if err != nil {
		return "", errors.New("failed to marshal chunks to JSON: " + err.Error())
	}

	chunksJsonFileName := filepath.Base(pdfPath) + ".chunks.json"
	chunksUrl, err := s.cloud.UploadStream(ctx, s.ccfg.SearchIndexBucket, tenant+"/"+chunksJsonFileName, []byte(chunksJson))
	if err != nil {
		return "", errors.New("failed to upload chunks JSON: " + err.Error())
	}

	return chunksUrl, nil
}
