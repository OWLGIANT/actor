package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/example/microshop/gateway"
	"github.com/example/microshop/pkg/config"
	"github.com/example/microshop/pkg/discovery"
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

	logger.Info("Starting API Gateway",
		zap.Int("port", cfg.Gateway.Port),
		zap.String("host", cfg.Gateway.Host))

	// Setup service discovery
	sd, err := discovery.NewServiceDiscovery(&cfg.Etcd)
	if err != nil {
		logger.Warn("Failed to connect to etcd, continuing without service discovery", zap.Error(err))
	}

	// Create gateway
	gw := gateway.NewGateway(cfg, logger, sd)
	gw.SetupRoutes()

	// Start gateway in goroutine
	gwErr := make(chan error, 1)
	go func() {
		if err := gw.Start(); err != nil {
			gwErr <- err
		}
	}()

	logger.Info("Gateway started successfully")

	// Wait for interrupt signal
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

	select {
	case <-sigCh:
		logger.Info("Received shutdown signal")
	case err := <-gwErr:
		logger.Fatal("Gateway error", zap.Error(err))
	}

	if sd != nil {
		sd.Close()
	}

	logger.Info("Gateway stopped")
}
