package activities

import (
	"context"

	"github.com/SaiNageswarS/agent-boot/search-core/appconfig"
	"github.com/SaiNageswarS/agent-boot/search-core/db"
	"github.com/SaiNageswarS/go-api-boot/cloud"
	"github.com/SaiNageswarS/go-api-boot/logger"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.uber.org/zap"
)

type InitTenantActivities struct {
	ccfg  *appconfig.AppConfig
	mongo *mongo.Client
	az    *cloud.Azure
}

func ProvideInitTenantActivities(ccfg *appconfig.AppConfig, az *cloud.Azure, mongo *mongo.Client) *InitTenantActivities {
	return &InitTenantActivities{
		ccfg:  ccfg,
		az:    az,
		mongo: mongo,
	}
}

func (s *InitTenantActivities) InitTenant(ctx context.Context, tenant string) error {
	// Initialize DB.
	if err := db.InitSearchCoreDB(ctx, s.mongo, tenant); err != nil {
		logger.Error("Failed to initialize search core DB", zap.String("tenant", tenant), zap.Error(err))
		return err
	}
	logger.Info("Search core DB initialized", zap.String("tenant", tenant))

	// Initialize Azure Blob Storage bucket for the tenant
	if err := s.az.EnsureBucket(ctx, tenant); err != nil {
		logger.Error("Failed to ensure Azure Container", zap.String("tenant", tenant), zap.Error(err))
		return err
	}

	logger.Info("Azure Container ensured", zap.String("tenant", tenant))
	return nil
}
