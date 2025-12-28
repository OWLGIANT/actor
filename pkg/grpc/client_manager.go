package grpc

import (
	"context"
	"fmt"
	"time"

	"github.com/example/microshop/pkg/config"
	"github.com/example/microshop/pkg/discovery"
	"github.com/example/microshop/pkg/proto"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// ClientManager manages gRPC client connections to microservices
type ClientManager struct {
	config    *config.Config
	discovery *discovery.ServiceDiscovery
	logger    *zap.Logger

	// gRPC clients
	userClient  proto.UserServiceClient
	orderClient proto.OrderServiceClient

	// gRPC connections
	userConn  *grpc.ClientConn
	orderConn *grpc.ClientConn
}

// NewClientManager creates a new gRPC client manager
func NewClientManager(cfg *config.Config, logger *zap.Logger, disc *discovery.ServiceDiscovery) *ClientManager {
	return &ClientManager{
		config:    cfg,
		discovery: disc,
		logger:    logger,
	}
}

// Connect establishes connections to all microservices
func (m *ClientManager) Connect() error {
	// Connect to User Service
	if err := m.connectUserService(); err != nil {
		return fmt.Errorf("failed to connect to user service: %w", err)
	}

	// Connect to Order Service
	if err := m.connectOrderService(); err != nil {
		return fmt.Errorf("failed to connect to order service: %w", err)
	}

	return nil
}

// connectUserService establishes a connection to the user service
func (m *ClientManager) connectUserService() error {
	// Default user service address
	target := "localhost:50051"

	// Try to use service discovery if available
	if m.discovery != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		instances, err := m.discovery.Discover(ctx, "user-service")
		if err == nil && len(instances) > 0 {
			target = instances[0].Host
			m.logger.Info("Discovered user service", zap.String("address", target))
		} else {
			m.logger.Info("Using default address for user service", zap.String("address", target))
		}
	}

	m.logger.Info("Connecting to user service", zap.String("target", target))

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	conn, err := grpc.DialContext(ctx, target,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
	)
	if err != nil {
		return err
	}

	m.userConn = conn
	m.userClient = proto.NewUserServiceClient(conn)

	m.logger.Info("Successfully connected to user service")
	return nil
}

// connectOrderService establishes a connection to the order service
func (m *ClientManager) connectOrderService() error {
	// Default order service address
	target := "localhost:50052"

	// Try to use service discovery if available
	if m.discovery != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		instances, err := m.discovery.Discover(ctx, "order-service")
		if err == nil && len(instances) > 0 {
			target = instances[0].Host
			m.logger.Info("Discovered order service", zap.String("address", target))
		} else {
			m.logger.Info("Using default address for order service", zap.String("address", target))
		}
	}

	m.logger.Info("Connecting to order service", zap.String("target", target))

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	conn, err := grpc.DialContext(ctx, target,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
	)
	if err != nil {
		return err
	}

	m.orderConn = conn
	m.orderClient = proto.NewOrderServiceClient(conn)

	m.logger.Info("Successfully connected to order service")
	return nil
}

// UserClient returns the user service gRPC client
func (m *ClientManager) UserClient() proto.UserServiceClient {
	return m.userClient
}

// OrderClient returns the order service gRPC client
func (m *ClientManager) OrderClient() proto.OrderServiceClient {
	return m.orderClient
}

// Close closes all gRPC connections
func (m *ClientManager) Close() error {
	var errs []error

	if m.userConn != nil {
		if err := m.userConn.Close(); err != nil {
			errs = append(errs, fmt.Errorf("user connection close error: %w", err))
		}
	}

	if m.orderConn != nil {
		if err := m.orderConn.Close(); err != nil {
			errs = append(errs, fmt.Errorf("order connection close error: %w", err))
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("errors closing connections: %v", errs)
	}

	return nil
}
