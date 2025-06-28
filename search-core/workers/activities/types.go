package activities

import (
	"context"
	"encoding/json"
	"errors"
	"os"

	"github.com/SaiNageswarS/agent-boot/search-core/appconfig"
	"github.com/SaiNageswarS/go-api-boot/cloud"
	"github.com/SaiNageswarS/go-api-boot/llm"
	"github.com/SaiNageswarS/go-api-boot/odm"
)

type Activities struct {
	ccfg *appconfig.AppConfig
	az   cloud.Cloud

	embedder llm.Embedder
	claude   *llm.AnthropicClient

	mongo odm.MongoClient
}

func ProvideActivities(ccfg *appconfig.AppConfig, az cloud.Cloud, embedder llm.Embedder, claude *llm.AnthropicClient, mongo odm.MongoClient) *Activities {
	return &Activities{
		ccfg:     ccfg,
		az:       az,
		embedder: embedder,
		claude:   claude,
		mongo:    mongo,
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
func writeToStorage(ctx context.Context, az cloud.Cloud, bucket, filePath string, data interface{}) (string, error) {
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
