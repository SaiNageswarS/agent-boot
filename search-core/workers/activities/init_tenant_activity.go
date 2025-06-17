package activities

import (
	"context"

	"github.com/SaiNageswarS/agent-boot/search-core/db"
	"github.com/SaiNageswarS/go-api-boot/logger"
	"go.uber.org/zap"
)

func (s *Activities) InitTenant(ctx context.Context, tenant string) error {
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
