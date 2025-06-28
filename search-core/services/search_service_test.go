package services

import (
	"strings"
	"testing"

	"github.com/SaiNageswarS/agent-boot/search-core/db"
	"github.com/SaiNageswarS/go-api-boot/dotenv"
	"github.com/SaiNageswarS/go-api-boot/llm"
	"github.com/SaiNageswarS/go-api-boot/odm"
	"github.com/SaiNageswarS/go-collection-boot/async"
	"github.com/SaiNageswarS/go-collection-boot/linq"
	"github.com/stretchr/testify/assert"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

func TestVectorSearch(t *testing.T) {
	dotenv.LoadEnv("../.env")

	mongoClient := odm.ProvideMongoClient()
	embedder := llm.ProvideJinaAIEmbeddingClient()

	testTenant := "devinderhealthcare"
	testQuery := "homeopathic remedies fear of death anxiety treatment"
	expectedChunkPrefixes := []string{"1544328200c1", "9a24dcec7d80"}

	t.Run("TestVectorSearch", func(t *testing.T) {
		queryVector, err := async.Await(embedder.GetEmbedding(t.Context(), testQuery, llm.WithTask("retrieval.query")))
		assert.NoError(t, err, "Failed to embed query")

		vecSearchTask := odm.CollectionOf[db.ChunkAnnModel](mongoClient, testTenant).
			VectorSearch(t.Context(), queryVector, odm.VectorSearchParams{
				IndexName:     db.VectorIndexName,
				Path:          db.VectorPath,
				K:             vecK,
				NumCandidates: 100,
			})

		vecSearchResults, err := async.Await(vecSearchTask)
		assert.NoError(t, err, "Failed to perform vector search")

		selectedChunkIds := linq.Map(vecSearchResults, func(c odm.SearchHit[db.ChunkAnnModel]) string {
			return c.Doc.ChunkID
		})

		assert.NotEmpty(t, selectedChunkIds, "Vector search should return results")

		for _, prefix := range expectedChunkPrefixes {
			found := linq.From(selectedChunkIds).Any(func(id string) bool {
				return strings.HasPrefix(id, prefix)
			})
			assert.True(t, found, "Expected chunk ID with prefix %s not found in results", prefix)
		}
	})

	t.Run("TestTextSearch", func(t *testing.T) {
		textSearchTask := odm.CollectionOf[db.ChunkModel](mongoClient, testTenant).
			TermSearch(t.Context(), testQuery, odm.TermSearchParams{
				IndexName: db.TextSearchIndexName,
				Path:      db.TextSearchPaths,
				Limit:     textK,
			})

		textSearchResults, err := async.Await(textSearchTask)
		assert.NoError(t, err, "Failed to perform text search")

		assert.NotEmpty(t, textSearchResults, "Text search should return results")
		selectedChunkIds := linq.Map(textSearchResults, func(c odm.SearchHit[db.ChunkModel]) string {
			return c.Doc.ChunkID
		})

		for _, prefix := range expectedChunkPrefixes {
			found := linq.From(selectedChunkIds).Any(func(id string) bool {
				return strings.HasPrefix(id, prefix)
			})
			assert.True(t, found, "Expected chunk ID with prefix %s not found in text search results", prefix)
		}
	})

	t.Run("TestHybridSearch", func(t *testing.T) {
		searchService := ProvideSearchService(mongoClient.(*mongo.Client), embedder)
		hybridSearchTask := searchService.hybridSearch(t.Context(), testTenant, testQuery)

		hybridSearchResults, err := async.Await(hybridSearchTask)
		assert.NoError(t, err, "Failed to perform hybrid search")

		assert.NotEmpty(t, hybridSearchResults, "Hybrid search should return results")
		selectedChunkIds := linq.Map(hybridSearchResults, func(c *db.ChunkModel) string {
			return c.ChunkID
		})

		for _, prefix := range expectedChunkPrefixes {
			found := linq.From(selectedChunkIds).Any(func(id string) bool {
				return strings.HasPrefix(id, prefix)
			})
			assert.True(t, found, "Expected chunk ID with prefix %s not found in hybrid search results", prefix)
		}
	})
}
