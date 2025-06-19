package activities

import (
	"context"
	"encoding/json"
	"errors"
	"os"

	"github.com/SaiNageswarS/agent-boot/search-core/appconfig"
	"github.com/SaiNageswarS/go-api-boot/cloud"
	"github.com/SaiNageswarS/go-api-boot/llm"
	"github.com/SaiNageswarS/go-api-boot/logger"
	"github.com/ollama/ollama/api"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.uber.org/zap"
)

type Activities struct {
	ccfg *appconfig.AppConfig
	az   *cloud.Azure

	ollamaClient *api.Client
	claude       *llm.AnthropicClient

	mongo *mongo.Client
}

func ProvideActivities(ccfg *appconfig.AppConfig, az *cloud.Azure, ollamaClient *api.Client, claude *llm.AnthropicClient, mongo *mongo.Client) *Activities {
	if err := az.EnsureBlob(context.Background()); err != nil {
		logger.Fatal("Failed to ensure Azure Blob Client", zap.Error(err))
	}

	return &Activities{
		ccfg:         ccfg,
		az:           az,
		ollamaClient: ollamaClient,
		claude:       claude,
		mongo:        mongo,
	}
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
