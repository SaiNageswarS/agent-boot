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

		chunksWithNeighbors := searchService.addNeighborsAndReorder(t.Context(), testTenant, hybridSearchResults)
		assert.NotEmpty(t, chunksWithNeighbors, "Chunks with neighbors should not be empty")
		assert.Len(t, chunksWithNeighbors, 30, "Chunks with neighbors should be more than or equal to hybrid search results")
	})
}
