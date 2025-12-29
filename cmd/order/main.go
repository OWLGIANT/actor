package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/example/microshop/pkg/config"
	"github.com/example/microshop/pkg/discovery"
	"github.com/example/microshop/pkg/grpc"
	"go.uber.org/zap"
)

func main() {
	// Load config
	cfg, err := config.Load("config/order-config.yaml")
	if err != nil {
		panic(fmt.Sprintf("Failed to load config: %v", err))
	}

	// Setup logger
	logger, err := zap.NewProduction()
	if err != nil {
		panic(fmt.Sprintf("Failed to create logger: %v", err))
	}
	defer logger.Sync()

	logger.Info("Starting order service",
		zap.String("name", cfg.Server.Name),
		zap.Int("port", cfg.Server.Port))

	// Create server
	server, err := grpc.NewOrderServer(cfg, logger)
	if err != nil {
		logger.Fatal("Failed to create server", zap.Error(err))
	}
	defer server.Close()

	// Connect to etcd for service discovery
	sd, err := discovery.NewServiceDiscovery(&cfg.Etcd)
	if err != nil {
		logger.Fatal("Failed to connect to etcd", zap.Error(err))
	}
	defer sd.Close()

	// Register service
	ctx := context.Background()
	err = sd.Register(ctx, &discovery.ServiceInstance{
		Name: cfg.Server.Name,
		Host: cfg.Server.Host,
		Port: cfg.Server.Port,
	})
	if err != nil {
		logger.Fatal("Failed to register service", zap.Error(err))
	}

	logger.Info("Service registered in etcd",
		zap.String("name", cfg.Server.Name),
		zap.String("address", fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)))

	// Ping dependencies
	if err := server.Redis().Ping(ctx); err != nil {
		logger.Warn("Redis connection failed", zap.Error(err))
	} else {
		logger.Info("Redis connected successfully")
	}

	// Start server in goroutine
	serverErr := make(chan error, 1)
	go func() {
		if err := server.Start(); err != nil {
			serverErr <- err
		}
	}()

	// Wait for interrupt signal
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

	select {
	case <-sigCh:
		logger.Info("Received shutdown signal")
	case err := <-serverErr:
		logger.Fatal("Server error", zap.Error(err))
	}

	// Deregister service
	if err := sd.Deregister(ctx, &discovery.ServiceInstance{
		Name: cfg.Server.Name,
		Host: cfg.Server.Host,
		Port: cfg.Server.Port,
	}); err != nil {
		logger.Error("Failed to deregister service", zap.Error(err))
	}

	logger.Info("Service stopped")
}
