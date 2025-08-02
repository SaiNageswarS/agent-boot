package db

import (
	"github.com/SaiNageswarS/go-api-boot/odm"
	"go.mongodb.org/mongo-driver/v2/bson"
)

const VectorIndexName = "chunkEmbeddingIndex"
const VectorPath = "embedding"

const EmbeddingDimensions = 2048 // jina ai 4

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
			Name:          VectorIndexName,
			Path:          VectorPath,
			Type:          "vector",
			NumDimensions: EmbeddingDimensions,
			Similarity:    "cosine",
			Quantization:  "scalar",
		},
	}
}
