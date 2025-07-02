package db

import (
	"context"

	"github.com/SaiNageswarS/go-api-boot/odm"
)

func InitSearchCoreDB(ctx context.Context, mongo odm.MongoClient, tenant string) error {
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

	err = odm.EnsureIndexes[AgentModel](ctx, mongo, tenant)
	if err != nil {
		return err
	}

	err = odm.EnsureIndexes[SessionModel](ctx, mongo, tenant)
	if err != nil {
		return err
	}

	return nil
}
