package db

import (
	"github.com/SaiNageswarS/go-api-boot/odm"
	"go.mongodb.org/mongo-driver/v2/bson"
)

type ChunkModel struct {
	ChunkID      string            `json:"chunkId" bson:"_id"`
	SectionPath  string            `json:"sectionPath" bson:"sectionPath"`
	SectionIndex int               `json:"sectionIndex" bson:"sectionIndex"` // Index of the section in the path
	Embedding    bson.Vector       `json:"-" bson:"embedding"`               // Embedding vector for the chunk, not serialized in JSON
	PHIRemoved   bool              `json:"phiRemoved" bson:"phiRemoved"`     // true if PHI is removed, false otherwise
	SourceURI    string            `json:"sourceUri" bson:"sourceUri"`       // e.g., "file://path/to/file.pdf"
	Body         string            `json:"body" bson:"body"`                 // The actual content of the chunk
	Tags         []string          `json:"tags" bson:"tags"`                 // Tags associated with the chunk
	Abbrevations map[string]string `json:"abbrevations" bson:"abbrevations"` // Abbreviations used in the chunk
}

func (m ChunkModel) Id() string { return m.ChunkID }

func (m ChunkModel) CollectionName() string { return "chunks" }

// Indexes
func (m ChunkModel) VectorIndexSpecs() []odm.VectorIndexSpec {
	return []odm.VectorIndexSpec{
		{
			Name:          "chunkEmbeddingIndex",
			Path:          "body",
			Type:          "vector",
			NumDimensions: 1024,
			Similarity:    "cosine",
			Quantization:  "scalar",
		},
	}
}

func (m ChunkModel) TermSearchIndexSpecs() []odm.TermSearchIndexSpec {
	return []odm.TermSearchIndexSpec{
		{
			Name:  "chunkIndex",
			Paths: []string{"body", "sectionPath", "tags"},
		},
	}
}
