package workers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"

	"github.com/SaiNageswarS/agent-boot/search-core/appconfig"
	"github.com/SaiNageswarS/agent-boot/search-core/db"
	"github.com/SaiNageswarS/go-api-boot/async"
	"github.com/SaiNageswarS/go-api-boot/cloud"
	"github.com/SaiNageswarS/go-api-boot/llm"
	"github.com/SaiNageswarS/go-api-boot/logger"
	"github.com/SaiNageswarS/go-api-boot/odm"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.uber.org/zap"
)

type Activities struct {
	ccfg     *appconfig.AppConfig
	az       *cloud.Azure
	chunker  *MarkdownChunker
	embedder *llm.JinaAIEmbeddingClient
	mongo    *mongo.Client
}

func ProvideIndexerActivities(ccfg *appconfig.AppConfig, az *cloud.Azure, llmClient *llm.AnthropicClient, embedder *llm.JinaAIEmbeddingClient, mongo *mongo.Client) *Activities {
	if err := az.EnsureBlob(context.Background()); err != nil {
		logger.Fatal("Failed to ensure Azure Blob Client", zap.Error(err))
	}

	return &Activities{
		ccfg:     ccfg,
		az:       az,
		embedder: embedder,
		chunker:  ProvideMarkdownChunker(llmClient),
		mongo:    mongo,
	}
}

// ChunkMarkdown processes a markdown file, chunks it into sections, and uploads the chunks to Azure Blob Storage.
// It returns the paths to the uploaded chunks JSON file. Each chunk is uploaded as a separate file in the specified {tenant}/{markdownFile} directory.
func (s *Activities) ChunkMarkdown(ctx context.Context, tenant, sourceUri, markdownFile, sectionsOutputPath string) ([]string, error) {
	markDownBytes, err := getBytes(s.az.DownloadFile(ctx, tenant, markdownFile))
	if err != nil {
		return []string{}, errors.New("failed to download PDF file: " + err.Error())
	}

	// Chunk the Markdown file
	chunks, err := s.chunker.ChunkMarkdownSections(ctx, sourceUri, markDownBytes)
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

func (s *Activities) EmbedAndStoreChunk(ctx context.Context, tenant, chunkPath string) error {
	// Download the chunk data
	chunkData, err := getBytes(s.az.DownloadFile(ctx, tenant, chunkPath))
	if err != nil {
		return errors.New("failed to download chunk file: " + err.Error())
	}

	chunkModel := db.ChunkModel{}
	err = json.Unmarshal(chunkData, &chunkModel)
	if err != nil {
		return errors.New("failed to unmarshal chunk data: " + err.Error())
	}

	// Embed the chunk using the LLM client
	embeddings, err := async.Await(s.embedder.GetEmbedding(ctx, llm.JinaAIEmbeddingRequest{Input: []string{chunkModel.SectionPath + " " + chunkModel.Body}}))
	if err != nil {
		return errors.New("failed to embed chunk: " + err.Error())
	}

	chunkModel.Embedding = bson.NewVector(embeddings)
	_, err = async.Await(odm.CollectionOf[db.ChunkModel](s.mongo, tenant).Save(ctx, chunkModel))
	if err != nil {
		return errors.New("failed to save chunk to database: " + err.Error())
	}

	return nil
}

func (s *Activities) InitTenant(ctx context.Context, tenant string) error {
	// Initialize DB.
	if err := db.InitSearchCoreDB(ctx, s.mongo, tenant); err != nil {
		logger.Error("Failed to initialize search core DB", zap.String("tenant", tenant), zap.Error(err))
		return err
	}

	// Initialize Azure Blob Storage bucket for the tenant
	if err := s.az.EnsureBucket(ctx, tenant); err != nil {
		logger.Error("Failed to ensure Azure Container", zap.String("tenant", tenant), zap.Error(err))
		return err
	}

	return nil
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
