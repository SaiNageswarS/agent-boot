package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	pb "agent-boot/proto/generated"

	"github.com/SaiNageswarS/agent-boot/search-core/appconfig"
	"github.com/SaiNageswarS/agent-boot/search-core/services"
	"github.com/SaiNageswarS/agent-boot/search-core/workers"
	"github.com/SaiNageswarS/go-api-boot/cloud"
	"github.com/SaiNageswarS/go-api-boot/config"
	"github.com/SaiNageswarS/go-api-boot/dotenv"
	"github.com/SaiNageswarS/go-api-boot/logger"
	"github.com/SaiNageswarS/go-api-boot/odm"
	"github.com/SaiNageswarS/go-api-boot/server"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	temporalClient "go.temporal.io/sdk/client"
	"go.uber.org/zap"
)

func main() {

	dotenv.LoadEnv()

	// load config file
	ccfgg := &appconfig.AppConfig{}
	err := config.LoadConfig("config.ini", ccfgg)
	if err != nil {
		logger.Fatal("Failed to load config", zap.Error(err))
	}

	az := cloud.ProvideAzure(&ccfgg.BootConfig)
	mongoClient, err := mongo.Connect(context.TODO(), options.Client().ApplyURI(ccfgg.MongoURI))
	if err != nil {
		logger.Fatal("Failed to connect to MongoDB", zap.Error(err))
	}

	boot, err := server.New().
		GRPCPort(":50051"). // or ":0" for dynamic
		HTTPPort(":8080").
		Provide(ccfgg).
		ProvideAs(az, (*cloud.Cloud)(nil)).
		ProvideAs(mongoClient, (*odm.MongoClient)(nil)).
		// Add Workers
		WithTemporal("search-core", &temporalClient.Options{
			HostPort: ccfgg.TemporalHostPort,
		}).
		RegisterTemporalActivity(workers.ProvideIndexerActivities).
		RegisterTemporalWorkflow(workers.IndexFileWorkflow).
		// Register gRPC service impls
		RegisterService(server.Adapt(pb.RegisterLoginServer), services.ProvideLoginService).
		Build()

	if err != nil {
		logger.Fatal("Dependency Injection Failed", zap.Error(err))
	}

	ctx := getCancellableContext()
	// catch SIGINT ‑> cancel
	_ = boot.Serve(ctx)
}

func getCancellableContext() context.Context {
	ctx, cancel := context.WithCancel(context.Background())

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sig
		cancel()
	}()

	return ctx
}
