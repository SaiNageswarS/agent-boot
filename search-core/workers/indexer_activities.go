package workers

import (
	"context"
	"encoding/json"
	"errors"
	"path/filepath"

	"github.com/SaiNageswarS/agent-boot/search-core/appconfig"
	"github.com/SaiNageswarS/go-api-boot/cloud"
)

type IndexerActivities struct {
	ccfg    *appconfig.AppConfig
	cloud   cloud.Cloud
	chunker *Chunker
}

func ProvideIndexerActivities(ccfg *appconfig.AppConfig, cloud cloud.Cloud) *IndexerActivities {
	return &IndexerActivities{
		ccfg:    ccfg,
		cloud:   cloud,
		chunker: ProvideChunker(),
	}
}

func (s *IndexerActivities) ChunkPDF(ctx context.Context, pdfUrl string) (string, error) {
	// Download the PDF file to a temporary location
	pdfPath, err := s.cloud.DownloadFile(ctx, s.ccfg.SearchIndexBucket, pdfUrl)
	if err != nil {
		return "", errors.New("failed to download PDF file: " + err.Error())
	}

	// Chunk the PDF file
	chunks, err := s.chunker.ChunkPDF(ctx, pdfPath)
	if err != nil {
		return "", errors.New("failed to chunk PDF file: " + err.Error())
	}

	// Upload the chunks JSON to the cloud
	jsonOut, err := json.MarshalIndent(chunks, "", "  ")
	if err != nil {
		return "", errors.New("failed to marshal chunks JSON: " + err.Error())
	}

	chunksJsonFileName := filepath.Base(pdfPath) + ".chunks.json"
	chunksUrl, err := s.cloud.UploadStream(ctx, s.ccfg.SearchIndexBucket, chunksJsonFileName, jsonOut)
	if err != nil {
		return "", errors.New("failed to upload chunks JSON: " + err.Error())
	}

	return chunksUrl, nil
}
