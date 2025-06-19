package db

import (
	"context"

	"github.com/SaiNageswarS/go-api-boot/odm"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

func InitSearchCoreDB(ctx context.Context, mongo *mongo.Client, tenant string) error {
	err := odm.EnsureIndexes[LoginModel](ctx, mongo, tenant)
	if err != nil {
		return err
	}

	err = odm.EnsureIndexes[ChunkModel](ctx, mongo, tenant)
	if err != nil {
		return err
	}

	err = odm.EnsureIndexes[ChunkAnnModel](ctx, mongo, tenant)
	if err != nil {
		return err
	}

	return nil
}
