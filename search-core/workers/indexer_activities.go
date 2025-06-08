package workers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"

	"github.com/SaiNageswarS/agent-boot/search-core/appconfig"
	"github.com/SaiNageswarS/go-api-boot/cloud"
	"github.com/SaiNageswarS/go-api-boot/llm"
	"github.com/SaiNageswarS/go-api-boot/logger"
	"go.uber.org/zap"
)

type IndexerActivities struct {
	ccfg    *appconfig.AppConfig
	az      *cloud.Azure
	chunker *MarkdownChunker
}

func ProvideIndexerActivities(ccfg *appconfig.AppConfig, az *cloud.Azure, llmClient *llm.AnthropicClient) *IndexerActivities {
	if err := az.EnsureBlob(context.Background()); err != nil {
		logger.Fatal("Failed to ensure Azure Blob Client", zap.Error(err))
	}

	return &IndexerActivities{
		ccfg:    ccfg,
		az:      az,
		chunker: ProvideMarkdownChunker(llmClient),
	}
}

// ChunkMarkdown processes a markdown file, chunks it into sections, and uploads the chunks to Azure Blob Storage.
// It returns the paths to the uploaded chunks JSON file. Each chunk is uploaded as a separate file in the specified {tenant}/{markdownFile} directory.
func (s *IndexerActivities) ChunkMarkdown(ctx context.Context, tenant, markdownFile, sectionsOutputPath string) ([]string, error) {
	markDownBytes, err := getBytes(s.az.DownloadFile(ctx, tenant, markdownFile))
	if err != nil {
		return []string{}, errors.New("failed to download PDF file: " + err.Error())
	}

	// Chunk the PDF file
	chunks, err := s.chunker.ChunkMarkdownSections(ctx, markdownFile, markDownBytes)
	if err != nil {
		return []string{}, errors.New("failed to chunk PDF file: " + err.Error())
	}

	// write combined sections to a single file for debugging purposes
	combinedSectionsFile := fmt.Sprintf("%s/combined_sections.json", sectionsOutputPath)
	writeToStorage(ctx, s.az, tenant, combinedSectionsFile, chunks)

	var sectionChunkPaths []string
	for _, chunk := range chunks {
		chunkPath := fmt.Sprintf("%s/%s.chunk.json", sectionsOutputPath, chunk.ChunkID)
		writeToStorage(ctx, s.az, tenant, chunkPath, chunk)
		sectionChunkPaths = append(sectionChunkPaths, chunkPath)
	}

	return sectionChunkPaths, nil
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

// Each tenant has separate bucket in Azure Blob Storage.
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
