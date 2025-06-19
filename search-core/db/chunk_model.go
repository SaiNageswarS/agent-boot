package db

import (
	"github.com/SaiNageswarS/go-api-boot/odm"
)

type ChunkModel struct {
	ChunkID      string            `json:"chunkId" bson:"_id"`
	Title        string            `json:"title" bson:"title"` // Title of the document, e.g., "Introduction to AI"
	SectionPath  string            `json:"sectionPath" bson:"sectionPath"`
	SectionIndex int               `json:"sectionIndex" bson:"sectionIndex"` // Index of the section in the path
	SourceURI    string            `json:"sourceUri" bson:"sourceUri"`       // e.g., "file://path/to/file.pdf"
	Tags         []string          `json:"tags" bson:"tags"`                 // Tags associated with the chunk
	Abbrevations map[string]string `json:"abbrevations" bson:"abbrevations"` // Abbreviations used in the chunk
	Sentences    []string          `json:"sentences" bson:"sentences"`       // Sentences in the chunk, used for text search
}

func (m ChunkModel) Id() string { return m.ChunkID }

func (m ChunkModel) CollectionName() string { return "chunks" }

// Indexes
func (m ChunkModel) TermSearchIndexSpecs() []odm.TermSearchIndexSpec {
	return []odm.TermSearchIndexSpec{
		{
			Name:  "chunkIndex",
			Paths: []string{"sentences", "sectionPath", "tags", "title"},
		},
	}
}
