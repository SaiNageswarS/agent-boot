package activities

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/SaiNageswarS/agent-boot/example/db"
	"github.com/SaiNageswarS/go-api-boot/odm"
	"github.com/SaiNageswarS/go-collection-boot/async"
)

func (s *Activities) SaveChunks(ctx context.Context, tenant string, chunkPaths []string) error {
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

		_, err = async.Await(odm.CollectionOf[db.ChunkModel](s.mongo, tenant).Save(ctx, chunkModel))
		if err != nil {
			return errors.New("failed to save chunk to database: " + err.Error())
		}
	}

	return nil
}
