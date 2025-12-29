package grpc

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"time"

	"go.mongodb.org/mongo-driver/bson"

	"github.com/example/microshop/pkg/config"
	"github.com/example/microshop/pkg/models"
	pb "github.com/example/microshop/pkg/proto"
	"github.com/example/microshop/pkg/repository"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/reflection"
	"google.golang.org/grpc/status"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

type OrderServer struct {
	pb.UnimplementedOrderServiceServer
	db     *gorm.DB
	redis  *repository.RedisRepository
	mongo  *repository.MongoRepository
	logger *zap.Logger
	config *config.Config
}

func NewOrderServer(cfg *config.Config, logger *zap.Logger) (*OrderServer, error) {
	// Connect to MySQL
	db, err := gorm.Open(mysql.Open(cfg.MySQL.DSN()), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to MySQL: %w", err)
	}

	// Auto migrate
	if err := db.AutoMigrate(&models.Order{}); err != nil {
		return nil, fmt.Errorf("failed to migrate: %w", err)
	}

	// Redis
	redisRepo := repository.NewRedisRepository(&cfg.Redis)

	// MongoDB
	mongoRepo, err := repository.NewMongoRepository(&cfg.MongoDB)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to MongoDB: %w", err)
	}

	return &OrderServer{
		db:     db,
		redis:  redisRepo,
		mongo:  mongoRepo,
		logger: logger,
		config: cfg,
	}, nil
}

func (s *OrderServer) Start() error {
	addr := fmt.Sprintf("%s:%d", s.config.Server.Host, s.config.Server.Port)
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("failed to listen: %w", err)
	}

	srv := grpc.NewServer()
	pb.RegisterOrderServiceServer(srv, s)
	reflection.Register(srv)

	s.logger.Info("Order service started", zap.String("address", addr))

	return srv.Serve(lis)
}

func (s *OrderServer) CreateOrder(ctx context.Context, req *pb.CreateOrderRequest) (*pb.CreateOrderResponse, error) {
	// Calculate total amount
	var totalAmount float64
	for _, item := range req.Items {
		totalAmount += float64(item.Quantity) * item.Price
	}

	order := &models.Order{
		ID:          generateUUID(),
		UserID:      req.UserId,
		TotalAmount: totalAmount,
		Status:      "pending",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	// Convert items to JSON for storage
	itemsData := make([]models.OrderItem, len(req.Items))
	for i, item := range req.Items {
		itemsData[i] = models.OrderItem{
			ProductID:   item.ProductId,
			ProductName: item.ProductName,
			Quantity:    item.Quantity,
			Price:       item.Price,
		}
	}
	itemsJSON, err := json.Marshal(itemsData)
	if err != nil {
		return &pb.CreateOrderResponse{Error: err.Error()}, status.Error(codes.Internal, "failed to serialize items")
	}
	order.Items = string(itemsJSON)

	if err := s.db.WithContext(ctx).Create(order).Error; err != nil {
		s.logger.Error("Failed to create order", zap.Error(err))
		return &pb.CreateOrderResponse{Error: err.Error()}, status.Error(codes.Internal, "failed to create order")
	}

	// Cache in Redis
	s.redis.CacheOrder(ctx, &repository.OrderCache{
		ID:     order.ID,
		UserID: order.UserID,
		Status: order.Status,
	})

	// Audit log
	go s.mongo.CreateAuditLog(context.Background(), &repository.AuditLog{
		Service:  "order-service",
		Action:   "create_order",
		EntityID: order.ID,
		Data:     bson.M{"user_id": order.UserID, "total_amount": totalAmount},
	})

	return &pb.CreateOrderResponse{
		Order: &pb.Order{
			Id:          order.ID,
			UserId:      order.UserID,
			Items:       req.Items,
			TotalAmount: totalAmount,
			Status:      order.Status,
			CreatedAt:   order.CreatedAt.Unix(),
			UpdatedAt:   order.UpdatedAt.Unix(),
		},
	}, nil
}

func (s *OrderServer) GetOrder(ctx context.Context, req *pb.GetOrderRequest) (*pb.GetOrderResponse, error) {
	var order models.Order
	if err := s.db.WithContext(ctx).Where("id = ?", req.Id).First(&order).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return &pb.GetOrderResponse{Error: "order not found"}, status.Error(codes.NotFound, "order not found")
		}
		return &pb.GetOrderResponse{Error: err.Error()}, status.Error(codes.Internal, "failed to get order")
	}

	// Parse items from JSON
	var itemsData []models.OrderItem
	if err := json.Unmarshal([]byte(order.Items), &itemsData); err != nil {
		return &pb.GetOrderResponse{Error: err.Error()}, status.Error(codes.Internal, "failed to parse items")
	}

	// Convert items to proto format
	items := make([]*pb.OrderItem, len(itemsData))
	for i, item := range itemsData {
		items[i] = &pb.OrderItem{
			ProductId:   item.ProductID,
			ProductName: item.ProductName,
			Quantity:    item.Quantity,
			Price:       item.Price,
		}
	}

	return &pb.GetOrderResponse{
		Order: &pb.Order{
			Id:          order.ID,
			UserId:      order.UserID,
			Items:       items,
			TotalAmount: order.TotalAmount,
			Status:      order.Status,
			CreatedAt:   order.CreatedAt.Unix(),
			UpdatedAt:   order.UpdatedAt.Unix(),
		},
	}, nil
}

func (s *OrderServer) ListOrders(ctx context.Context, req *pb.ListOrdersRequest) (*pb.ListOrdersResponse, error) {
	var orders []models.Order
	var total int64

	query := s.db.WithContext(ctx).Model(&models.Order{})
	if req.UserId != "" {
		query = query.Where("user_id = ?", req.UserId)
	}
	query.Count(&total)

	offset := (int(req.Page) - 1) * int(req.PageSize)
	if err := query.Offset(offset).Limit(int(req.PageSize)).Find(&orders).Error; err != nil {
		return &pb.ListOrdersResponse{Error: err.Error()}, status.Error(codes.Internal, "failed to list orders")
	}

	pbOrders := make([]*pb.Order, len(orders))
	for i, o := range orders {
		// Parse items from JSON
		var itemsData []models.OrderItem
		if err := json.Unmarshal([]byte(o.Items), &itemsData); err != nil {
			s.logger.Warn("Failed to parse items for order", zap.String("order_id", o.ID), zap.Error(err))
			itemsData = []models.OrderItem{}
		}

		items := make([]*pb.OrderItem, len(itemsData))
		for j, item := range itemsData {
			items[j] = &pb.OrderItem{
				ProductId:   item.ProductID,
				ProductName: item.ProductName,
				Quantity:    item.Quantity,
				Price:       item.Price,
			}
		}
		pbOrders[i] = &pb.Order{
			Id:          o.ID,
			UserId:      o.UserID,
			Items:       items,
			TotalAmount: o.TotalAmount,
			Status:      o.Status,
			CreatedAt:   o.CreatedAt.Unix(),
			UpdatedAt:   o.UpdatedAt.Unix(),
		}
	}

	return &pb.ListOrdersResponse{
		Orders: pbOrders,
		Total:  int32(total),
	}, nil
}

func (s *OrderServer) UpdateOrderStatus(ctx context.Context, req *pb.UpdateOrderStatusRequest) (*pb.UpdateOrderStatusResponse, error) {
	var order models.Order
	if err := s.db.WithContext(ctx).Where("id = ?", req.OrderId).First(&order).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return &pb.UpdateOrderStatusResponse{Error: "order not found"}, status.Error(codes.NotFound, "order not found")
		}
		return &pb.UpdateOrderStatusResponse{Error: err.Error()}, status.Error(codes.Internal, "failed to update order")
	}

	updates := map[string]interface{}{
		"status":     req.Status,
		"updated_at": time.Now(),
	}

	if err := s.db.WithContext(ctx).Model(&order).Updates(updates).Error; err != nil {
		return &pb.UpdateOrderStatusResponse{Error: err.Error()}, status.Error(codes.Internal, "failed to update order")
	}

	// Invalidate cache
	s.redis.Del(ctx, fmt.Sprintf("order:%s", req.OrderId))

	// Parse items from JSON
	var itemsData []models.OrderItem
	if err := json.Unmarshal([]byte(order.Items), &itemsData); err != nil {
		return &pb.UpdateOrderStatusResponse{Error: err.Error()}, status.Error(codes.Internal, "failed to parse items")
	}

	// Convert items to proto format
	items := make([]*pb.OrderItem, len(itemsData))
	for i, item := range itemsData {
		items[i] = &pb.OrderItem{
			ProductId:   item.ProductID,
			ProductName: item.ProductName,
			Quantity:    item.Quantity,
			Price:       item.Price,
		}
	}

	return &pb.UpdateOrderStatusResponse{
		Order: &pb.Order{
			Id:          order.ID,
			UserId:      order.UserID,
			Items:       items,
			TotalAmount: order.TotalAmount,
			Status:      req.Status,
			CreatedAt:   order.CreatedAt.Unix(),
			UpdatedAt:   time.Now().Unix(),
		},
	}, nil
}

func (s *OrderServer) Close() error {
	s.redis.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return s.mongo.Close(ctx)
}

func (s *OrderServer) Redis() *repository.RedisRepository {
	return s.redis
}
