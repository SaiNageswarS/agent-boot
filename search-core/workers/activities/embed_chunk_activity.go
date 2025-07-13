package activities

import (
	"context"
	"errors"
	"strings"

	"github.com/SaiNageswarS/agent-boot/search-core/db"
	"github.com/SaiNageswarS/go-api-boot/llm"
	"github.com/SaiNageswarS/go-api-boot/logger"
	"github.com/SaiNageswarS/go-api-boot/odm"
	"github.com/SaiNageswarS/go-collection-boot/async"
	"github.com/SaiNageswarS/go-collection-boot/linq"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.uber.org/zap"
)

// Useful to introduce new embedding models or change the existing one.
func (s *Activities) GetChunksWithMissingEmbeddings(ctx context.Context, tenant, sourceUri string) ([]string, error) {
	if sourceUri == "" {
		return nil, errors.New("sourceUri cannot be empty")
	}

	// ---- 1. Find all chunks belonging to the sourceUri ------
	filter := bson.M{
		"sourceUri": sourceUri,
	}

	chunkModels, err := async.Await(odm.CollectionOf[db.ChunkModel](s.mongo, tenant).Find(ctx, filter, nil, 0, 0))
	if err != nil {
		return nil, errors.New("failed to find chunks with missing embeddings: " + err.Error())
	}
	if len(chunkModels) == 0 {
		return nil, nil // nothing to do
	}

	chunkIds := linq.Map(chunkModels, func(chunk db.ChunkModel) string {
		return chunk.ChunkID
	})

	logger.Info("Chunks found", zap.String("sourceUri", sourceUri), zap.Int("count", len(chunkIds)))

	// ---- 2. Find chunks that are missing embeddings ------
	annFilter := bson.M{
		"_id": bson.M{"$in": chunkIds},
	}
	chunkAnnModels, err := async.Await(odm.CollectionOf[db.ChunkAnnModel](s.mongo, tenant).Find(ctx, annFilter, nil, 0, 0))
	if err != nil {
		return nil, errors.New("failed to find chunk annotations: " + err.Error())
	}

	chunkAnnIdsPresent := make(map[string]bool, len(chunkAnnModels))
	for _, annModel := range chunkAnnModels {
		chunkAnnIdsPresent[annModel.ChunkID] = true
	}

	chunkIds = linq.From(chunkIds).
		Where(func(chunkId string) bool {
			// Check if the chunk ID is not present in the annotations
			return !chunkAnnIdsPresent[chunkId]
		}).
		ToSlice()

	if len(chunkIds) == 0 {
		logger.Info("No chunks with missing embeddings found", zap.String("sourceUri", sourceUri))
		return nil, nil // No chunks with missing embeddings
	}

	logger.Info("Chunks with missing embeddings found", zap.String("sourceUri", sourceUri), zap.Int("count", len(chunkIds)))
	return chunkIds, nil
}

func (s *Activities) EmbedChunks(ctx context.Context, tenant string, chunkIds []string) error {
	// Download the chunk data
	for idx, chunkId := range chunkIds {
		chunkModel, err := async.Await(odm.CollectionOf[db.ChunkModel](s.mongo, tenant).FindOneByID(ctx, chunkId))
		if err != nil {
			return errors.New("failed to find chunk by ID: " + err.Error())
		}

		// Embed the chunk using the LLM client
		embeddingText := chunkModel.SectionPath + "\n" + strings.Join(chunkModel.Sentences, "\n")

		embeddings, err := async.Await(s.embedder.GetEmbedding(ctx, embeddingText, llm.WithTask("retrieval.passage")))
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

		if idx%100 == 0 {
			logger.Info("Embedded chunks progress", zap.Int("processed", idx), zap.Int("total", len(chunkIds)), zap.String("chunkId", chunkId))
		}
	}

	return nil
}
