package workers

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"

	"github.com/SaiNageswarS/agent-boot/search-core/appconfig"
	"github.com/SaiNageswarS/go-api-boot/cloud"
	"github.com/SaiNageswarS/go-api-boot/logger"
	"go.uber.org/zap"
)

type IndexerActivities struct {
	ccfg    *appconfig.AppConfig
	az      *cloud.Azure
	chunker *MarkdownChunker
}

func ProvideIndexerActivities(ccfg *appconfig.AppConfig, az *cloud.Azure) *IndexerActivities {
	if err := az.EnsureBlob(context.Background()); err != nil {
		logger.Fatal("Failed to ensure Azure Blob Client", zap.Error(err))
	}

	return &IndexerActivities{
		ccfg:    ccfg,
		az:      az,
		chunker: ProvideMarkdownChunker(),
	}
}

func (s *IndexerActivities) ChunkMarkdown(ctx context.Context, tenant, markdownFile string) (string, error) {
	markDownBytes, err := getBytes(s.az.DownloadFile(ctx, s.ccfg.SearchIndexBucket, tenant+"/"+markdownFile))
	if err != nil {
		return "", errors.New("failed to download PDF file: " + err.Error())
	}

	// Chunk the PDF file
	chunks, err := s.chunker.ChunkMarkdownSections(ctx, markdownFile, markDownBytes)
	if err != nil {
		return "", errors.New("failed to chunk PDF file: " + err.Error())
	}

	// Upload the chunks JSON to the cloud
	return writeToStorage(ctx, s.az, s.ccfg.SearchIndexBucket, tenant+"/"+filepath.Base(markdownFile)+".chunks.json", chunks)
}

func getBytes(filePath string, err error) ([]byte, error) {
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, errors.New("failed to read file: " + err.Error())
	}
	return data, nil
}

func writeToStorage(ctx context.Context, az *cloud.Azure, bucket, filePath string, data interface{}) (string, error) {
	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return "", errors.New("failed to marshal data: " + err.Error())
	}

	// Upload the JSON data to the cloud
	_, err = az.UploadBuffer(ctx, bucket, filePath, jsonData)
	if err != nil {
		return "", errors.New("failed to upload JSON data: " + err.Error())
	}

	return filePath, nil
}
