package agent

import (
	"strings"
	"testing"

	"github.com/SaiNageswarS/agent-boot/example/db"
	"github.com/SaiNageswarS/go-api-boot/dotenv"
	"github.com/SaiNageswarS/go-api-boot/embed"
	"github.com/SaiNageswarS/go-api-boot/odm"
	"github.com/SaiNageswarS/go-collection-boot/ds"
	"github.com/SaiNageswarS/go-collection-boot/linq"
	"github.com/stretchr/testify/assert"
)

func TestSearch(t *testing.T) {
	dotenv.LoadEnv("../.env")

	mongoClient := odm.ProvideMongoClient()
	embedder := embed.ProvideJinaAIEmbeddingClient()

	testTenant := "devinderhealthcare"
	chunkRepository := odm.CollectionOf[db.ChunkModel](mongoClient, testTenant)
	vectorRepository := odm.CollectionOf[db.ChunkAnnModel](mongoClient, testTenant)

	testQuery := "homeopathic remedies fear of death anxiety treatment"
	expectedChunkPrefixes := []string{"1544328200c1", "9a24dcec7d80"}

	t.Run("TestHybridSearch", func(t *testing.T) {
		searchService := NewSearchStep(chunkRepository, vectorRepository, embedder)
		hybridSearchResults, err := searchService.Run(t.Context(), []string{testQuery})
		assert.NoError(t, err, "Failed to create hybrid search task")

		assert.NotEmpty(t, hybridSearchResults, "Hybrid search should return results")

		selectedSections := ds.NewSet[string]()
		_, err = linq.Pipe2(
			linq.FromSlice(t.Context(), hybridSearchResults),

			linq.Select(func(c *db.ChunkModel) string {
				chunkIdParts := strings.Split(c.ChunkID, "_")
				return chunkIdParts[0] // Extract the prefix from ChunkID
			}),

			linq.ForEach(func(section string) {
				selectedSections.Add(section)
			}),
		)

		assert.NoError(t, err, "Failed to process hybrid search results")

		for _, prefix := range expectedChunkPrefixes {
			assert.True(t, selectedSections.Contains(prefix), "Expected chunk ID with prefix %s not found in hybrid search results", prefix)
		}

		assert.True(t, len(hybridSearchResults) > 30, "Chunks with neighbors should be more than or equal to hybrid search results")
	})
}
