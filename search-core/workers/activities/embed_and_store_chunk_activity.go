package activities

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	"github.com/SaiNageswarS/agent-boot/search-core/db"
	"github.com/SaiNageswarS/agent-boot/search-core/prompts"
	"github.com/SaiNageswarS/go-api-boot/odm"
	"github.com/SaiNageswarS/go-collection-boot/async"
	"go.mongodb.org/mongo-driver/v2/bson"
)

func (s *Activities) EmbedAndStoreChunk(ctx context.Context, tenant string, chunkPaths []string) error {
	// Download the chunk data
	for _, chunkPath := range chunkPaths {
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
		embeddingText := chunkModel.Title + "\n" + chunkModel.SectionPath + "\n" + strings.Join(chunkModel.Sentences, "\n")

		embeddings, err := prompts.EmbedOnce(ctx, s.ollamaClient, embeddingText)
		if err != nil {
			return errors.New("failed to embed chunk: " + err.Error())
		}

		chunkModel.Embedding = bson.NewVector(embeddings)

		_, err = async.Await(odm.CollectionOf[db.ChunkModel](s.mongo, tenant).Save(ctx, chunkModel))
		if err != nil {
			return errors.New("failed to save chunk to database: " + err.Error())
		}
	}

	return nil
}
