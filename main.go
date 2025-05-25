package main

import (
	"context"

	"github.com/SaiNageswarS/agent-boot/generated/pb"
	"github.com/SaiNageswarS/agent-boot/services"
	"github.com/SaiNageswarS/go-api-boot/config"
	"github.com/SaiNageswarS/go-api-boot/dotenv"
	"github.com/SaiNageswarS/go-api-boot/logger"
	"github.com/SaiNageswarS/go-api-boot/odm"
	"github.com/SaiNageswarS/go-api-boot/server"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.uber.org/zap"
)

func main() {
	dotenv.LoadEnv()

	// load config file
	ccfgg := &config.BootConfig{}
	err := config.LoadConfig("config.ini", ccfgg)
	if err != nil {
		logger.Fatal("Failed to load config", zap.Error(err))
	}

	mongoClient, err := mongo.Connect(context.TODO(), options.Client().ApplyURI(ccfgg.MongoUri))
	if err != nil {
		logger.Fatal("Failed to connect to MongoDB", zap.Error(err))
	}

	boot, err := server.New().
		GRPCPort(":50051"). // or ":0" for dynamic
		HTTPPort(":8080").
		Provide(ccfgg).
		ProvideAs(mongoClient, (*odm.MongoClient)(nil)).
		// Register gRPC service impls
		Register(server.Adapt(pb.RegisterLoginServer), services.ProvideLoginService).
		Build()

	if err != nil {
		logger.Fatal("Dependency Injection Failed", zap.Error(err))
	}

	ctx, _ := context.WithCancel(context.Background())
	// catch SIGINT ‑> cancel
	_ = boot.Serve(ctx)
}
