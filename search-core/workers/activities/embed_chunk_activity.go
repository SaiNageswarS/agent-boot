package activities

import (
	"context"
	"errors"
	"strings"

	"github.com/SaiNageswarS/agent-boot/search-core/db"
	"github.com/SaiNageswarS/agent-boot/search-core/prompts"
	"github.com/SaiNageswarS/go-api-boot/odm"
	"github.com/SaiNageswarS/go-collection-boot/async"
	"go.mongodb.org/mongo-driver/v2/bson"
)

// Useful to introduce new embedding models or change the existing one.
func (s *Activities) GetChunksWithMissingEmbeddings(ctx context.Context, tenant, sourceUri string) ([]string, error) {
	if sourceUri == "" {
		return nil, errors.New("sourceUri cannot be empty")
	}

	// find all chunks belonging to the sourceUri.
	filter := bson.M{
		"sourceUri": sourceUri,
	}

	chunkModels, err := async.Await(odm.CollectionOf[db.ChunkModel](s.mongo, tenant).Find(ctx, filter, nil, 0, 0))
	if err != nil {
		return nil, errors.New("failed to find chunks with missing embeddings: " + err.Error())
	}

	var chunkIds []string
	for _, chunkModel := range chunkModels {
		chunkIds = append(chunkIds, chunkModel.ChunkID)
	}

	return chunkIds, nil
}

func (s *Activities) EmbedChunks(ctx context.Context, tenant string, chunkIds []string) error {
	// Download the chunk data
	for _, chunkId := range chunkIds {
		chunkModel, err := async.Await(odm.CollectionOf[db.ChunkModel](s.mongo, tenant).FindOneByID(ctx, chunkId))
		if err != nil {
			return errors.New("failed to find chunk by ID: " + err.Error())
		}

		// Embed the chunk using the LLM client
		embeddingText := chunkModel.Title + "\n" + chunkModel.SectionPath + "\n" + strings.Join(chunkModel.Sentences, "\n")

		embeddings, err := prompts.EmbedOnce(ctx, s.ollamaClient, embeddingText)
		if err != nil {
			return errors.New("failed to embed chunk: " + err.Error())
		}

		chunkAnn := db.ChunkAnnModel{
			ChunkID:   chunkModel.ChunkID,
			Embedding: bson.NewVector(embeddings),
		}

		_, err = async.Await(odm.CollectionOf[db.ChunkAnnModel](s.mongo, tenant).Save(ctx, chunkAnn))
		if err != nil {
			return errors.New("failed to save chunk to database: " + err.Error())
		}
	}

	return nil
}
