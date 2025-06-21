package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	pb "agent-boot/proto/generated"

	"github.com/SaiNageswarS/agent-boot/search-core/appconfig"
	"github.com/SaiNageswarS/agent-boot/search-core/services"
	"github.com/SaiNageswarS/agent-boot/search-core/workers/activities"
	"github.com/SaiNageswarS/agent-boot/search-core/workers/workflows"
	"github.com/SaiNageswarS/go-api-boot/cloud"
	"github.com/SaiNageswarS/go-api-boot/config"
	"github.com/SaiNageswarS/go-api-boot/dotenv"
	"github.com/SaiNageswarS/go-api-boot/llm"
	"github.com/SaiNageswarS/go-api-boot/logger"
	"github.com/SaiNageswarS/go-api-boot/odm"
	"github.com/SaiNageswarS/go-api-boot/server"
	"github.com/ollama/ollama/api"
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

	claude, err := llm.ProvideAnthropicClient()
	if err != nil {
		logger.Fatal("Failed to create Claude client", zap.Error(err))
	}

	ollamaClient, err := api.ClientFromEnvironment()
	if err != nil {
		logger.Fatal("Failed to create Ollama client", zap.Error(err))
	}

	mongoClient, err := odm.GetClient()
	if err != nil {
		logger.Fatal("Failed to connect to MongoDB", zap.Error(err))
	}

	boot, err := server.New().
		GRPCPort(":50051"). // or ":0" for dynamic
		HTTPPort(":8081").
		Provide(ccfgg).
		Provide(az).
		Provide(ollamaClient).
		Provide(claude).
		Provide(mongoClient).
		// Add Workers
		WithTemporal(ccfgg.TemporalGoTaskQueue, &temporalClient.Options{
			HostPort: ccfgg.TemporalHostPort,
		}).
		RegisterTemporalActivity(activities.ProvideActivities).
		RegisterTemporalWorkflow(workflows.ChunkMarkdownWorkflow).
		RegisterTemporalWorkflow(workflows.InitTenantWorkflow).
		RegisterTemporalWorkflow(workflows.PdfHandlerWorkflow).
		RegisterTemporalWorkflow(workflows.EmbedChunksWorkflow).
		// Register gRPC service impls
		RegisterService(server.Adapt(pb.RegisterLoginServer), services.ProvideLoginService).
		RegisterService(server.Adapt(pb.RegisterSearchServer), services.ProvideSearchService).
		RegisterService(server.Adapt(pb.RegisterAgentServer), services.ProvideAgentService).
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
