package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/example/microshop/cmd/actor"
	"github.com/example/microshop/pkg/config"
	"go.uber.org/zap"
)

func main() {
	// Load config
	cfg, err := config.Load("config/config.yaml")
	if err != nil {
		panic(fmt.Sprintf("Failed to load config: %v", err))
	}

	// Setup logger
	logger, err := zap.NewProduction()
	if err != nil {
		panic(fmt.Sprintf("Failed to create logger: %v", err))
	}
	defer logger.Sync()

	logger.Info("Starting ProtoActor service")

	// Start actor service
	if err := actor.StartActorService(cfg, logger); err != nil {
		logger.Fatal("Failed to start actor service", zap.Error(err))
	}

	// Wait for interrupt signal
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

	<-sigCh
	logger.Info("ProtoActor service stopped")
}
