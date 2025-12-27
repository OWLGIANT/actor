package grpc

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/example/microshop/pkg/config"
	"github.com/example/microshop/pkg/models"
	pb "github.com/example/microshop/proto/user"
	"github.com/example/microshop/pkg/repository"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/reflection"
	"google.golang.org/grpc/status"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

type UserServer struct {
	pb.UnimplementedUserServiceServer
	db      *gorm.DB
	redis   *repository.RedisRepository
	mongo   *repository.MongoRepository
	logger  *zap.Logger
	config  *config.Config
}

func NewUserServer(cfg *config.Config, logger *zap.Logger) (*UserServer, error) {
	// Connect to MySQL
	db, err := gorm.Open(mysql.Open(cfg.MySQL.DSN()), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to MySQL: %w", err)
	}

	// Auto migrate
	if err := db.AutoMigrate(&models.User{}); err != nil {
		return nil, fmt.Errorf("failed to migrate: %w", err)
	}

	// Redis
	redisRepo := repository.NewRedisRepository(&cfg.Redis)

	// MongoDB
	mongoRepo, err := repository.NewMongoRepository(&cfg.MongoDB)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to MongoDB: %w", err)
	}

	return &UserServer{
		db:     db,
		redis:  redisRepo,
		mongo:  mongoRepo,
		logger: logger,
		config: cfg,
	}, nil
}

func (s *UserServer) Start() error {
	addr := fmt.Sprintf("%s:%d", s.config.Server.Host, s.config.Server.Port)
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("failed to listen: %w", err)
	}

	srv := grpc.NewServer()
	pb.RegisterUserServiceServer(srv, s)
	reflection.Register(srv)

	s.logger.Info("User service started", zap.String("address", addr))

	return srv.Serve(lis)
}

func (s *UserServer) CreateUser(ctx context.Context, req *pb.CreateUserRequest) (*pb.CreateUserResponse, error) {
	user := &models.User{
		ID:        generateUUID(),
		Name:      req.Name,
		Email:     req.Email,
		Phone:     req.Phone,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := s.db.WithContext(ctx).Create(user).Error; err != nil {
		s.logger.Error("Failed to create user", zap.Error(err))
		return &pb.CreateUserResponse{Error: err.Error()}, status.Error(codes.Internal, "failed to create user")
	}

	// Cache in Redis
	s.redis.CacheUser(ctx, &repository.UserCache{
		ID:    user.ID,
		Name:  user.Name,
		Email: user.Email,
		Phone: user.Phone,
	})

	// Audit log
	go s.mongo.CreateAuditLog(context.Background(), &repository.AuditLog{
		Service:  "user-service",
		Action:   "create_user",
		EntityID: user.ID,
		Data:     bson.M{"name": user.Name, "email": user.Email},
	})

	return &pb.CreateUserResponse{
		User: &pb.User{
			Id:        user.ID,
			Name:      user.Name,
			Email:     user.Email,
			Phone:     user.Phone,
			CreatedAt: user.CreatedAt.Unix(),
			UpdatedAt: user.UpdatedAt.Unix(),
		},
	}, nil
}

func (s *UserServer) GetUser(ctx context.Context, req *pb.GetUserRequest) (*pb.GetUserResponse, error) {
	// Try cache first
	cachedUser, err := s.redis.GetUserCache(ctx, req.Id)
	if err == nil {
		return &pb.GetUserResponse{
			User: &pb.User{
				Id:    cachedUser.ID,
				Name:  cachedUser.Name,
				Email: cachedUser.Email,
				Phone: cachedUser.Phone,
			},
		}, nil
	}

	var user models.User
	if err := s.db.WithContext(ctx).Where("id = ?", req.Id).First(&user).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return &pb.GetUserResponse{Error: "user not found"}, status.Error(codes.NotFound, "user not found")
		}
		return &pb.GetUserResponse{Error: err.Error()}, status.Error(codes.Internal, "failed to get user")
	}

	// Update cache
	s.redis.CacheUser(ctx, &repository.UserCache{
		ID:    user.ID,
		Name:  user.Name,
		Email: user.Email,
		Phone: user.Phone,
	})

	return &pb.GetUserResponse{
		User: &pb.User{
			Id:        user.ID,
			Name:      user.Name,
			Email:     user.Email,
			Phone:     user.Phone,
			CreatedAt: user.CreatedAt.Unix(),
			UpdatedAt: user.UpdatedAt.Unix(),
		},
	}, nil
}

func (s *UserServer) ListUsers(ctx context.Context, req *pb.ListUsersRequest) (*pb.ListUsersResponse, error) {
	var users []models.User
	var total int64

	query := s.db.WithContext(ctx).Model(&models.User{})
	query.Count(&total)

	offset := (int(req.Page) - 1) * int(req.PageSize)
	if err := query.Offset(offset).Limit(int(req.PageSize)).Find(&users).Error; err != nil {
		return &pb.ListUsersResponse{Error: err.Error()}, status.Error(codes.Internal, "failed to list users")
	}

	pbUsers := make([]*pb.User, len(users))
	for i, u := range users {
		pbUsers[i] = &pb.User{
			Id:        u.ID,
			Name:      u.Name,
			Email:     u.Email,
			Phone:     u.Phone,
			CreatedAt: u.CreatedAt.Unix(),
			UpdatedAt: u.UpdatedAt.Unix(),
		}
	}

	return &pb.ListUsersResponse{
		Users: pbUsers,
		Total: int32(total),
	}, nil
}

func (s *UserServer) UpdateUser(ctx context.Context, req *pb.UpdateUserRequest) (*pb.UpdateUserResponse, error) {
	var user models.User
	if err := s.db.WithContext(ctx).Where("id = ?", req.Id).First(&user).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return &pb.UpdateUserResponse{Error: "user not found"}, status.Error(codes.NotFound, "user not found")
		}
		return &pb.UpdateUserResponse{Error: err.Error()}, status.Error(codes.Internal, "failed to update user")
	}

	updates := map[string]interface{}{
		"updated_at": time.Now(),
	}
	if req.Name != "" {
		updates["name"] = req.Name
	}
	if req.Email != "" {
		updates["email"] = req.Email
	}
	if req.Phone != "" {
		updates["phone"] = req.Phone
	}

	if err := s.db.WithContext(ctx).Model(&user).Updates(updates).Error; err != nil {
		return &pb.UpdateUserResponse{Error: err.Error()}, status.Error(codes.Internal, "failed to update user")
	}

	// Invalidate cache
	s.redis.Del(ctx, fmt.Sprintf("user:%s", req.Id))

	return &pb.UpdateUserResponse{
		User: &pb.User{
			Id:        user.ID,
			Name:      user.Name,
			Email:     user.Email,
			Phone:     user.Phone,
			CreatedAt: user.CreatedAt.Unix(),
			UpdatedAt: user.UpdatedAt.Unix(),
		},
	}, nil
}

func (s *UserServer) DeleteUser(ctx context.Context, req *pb.DeleteUserRequest) (*pb.DeleteUserResponse, error) {
	if err := s.db.WithContext(ctx).Delete(&models.User{}, "id = ?", req.Id).Error; err != nil {
		return &pb.DeleteUserResponse{Error: err.Error()}, status.Error(codes.Internal, "failed to delete user")
	}

	// Delete cache
	s.redis.Del(ctx, fmt.Sprintf("user:%s", req.Id))

	return &pb.DeleteUserResponse{
		Success: true,
	}, nil
}

func (s *UserServer) Close() error {
	s.redis.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return s.mongo.Close(ctx)
}

import (
	"net"

	"go.mongodb.org/mongo-driver/bson"
)

func generateUUID() string {
	return fmt.Sprintf("%d", time.Now().UnixNano())
}
