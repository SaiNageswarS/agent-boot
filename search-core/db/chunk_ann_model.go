package db

import (
	"github.com/SaiNageswarS/agent-boot/search-core/prompts"
	"github.com/SaiNageswarS/go-api-boot/odm"
	"go.mongodb.org/mongo-driver/v2/bson"
)

type ChunkAnnModel struct {
	ChunkID   string      `json:"chunkId" bson:"_id"` // Unique
	Embedding bson.Vector `json:"-" bson:"embedding"` // Embedding vector for the chunk, not serialized in JSON
}

func (m ChunkAnnModel) Id() string { return m.ChunkID }

func (m ChunkAnnModel) CollectionName() string { return "chunk_ann_index" }

// Indexes
func (m ChunkAnnModel) VectorIndexSpecs() []odm.VectorIndexSpec {
	return []odm.VectorIndexSpec{
		{
			Name:          "chunkEmbeddingIndex",
			Path:          "embedding",
			Type:          "vector",
			NumDimensions: prompts.EmbeddingDimensions,
			Similarity:    "cosine",
			Quantization:  "scalar",
		},
	}
}
