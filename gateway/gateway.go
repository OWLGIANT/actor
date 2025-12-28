package gateway

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/example/microshop/pkg/config"
	"github.com/example/microshop/pkg/discovery"
	"github.com/example/microshop/pkg/grpc"
	"github.com/example/microshop/pkg/proto"
	"github.com/gin-gonic/gin"
	ginSwagger "github.com/swaggo/gin-swagger"
	"github.com/swaggo/files"
	"go.uber.org/zap"
)

type Gateway struct {
	config        *config.Config
	discovery     *discovery.ServiceDiscovery
	logger        *zap.Logger
	router        *gin.Engine
	grpcClients   *grpc.ClientManager
}

func NewGateway(cfg *config.Config, logger *zap.Logger, disc *discovery.ServiceDiscovery) *Gateway {
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.Use(gin.Recovery())
	router.Use(loggerMiddleware(logger))

	// Create gRPC client manager
	grpcMgr := grpc.NewClientManager(cfg, logger, disc)

	return &Gateway{
		config:      cfg,
		discovery:   disc,
		logger:      logger,
		router:      router,
		grpcClients: grpcMgr,
	}
}

// Connect connects to all gRPC services
func (g *Gateway) Connect() error {
	return g.grpcClients.Connect()
}

func (g *Gateway) SetupRoutes() {
	// Health check
	g.router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// API v1 routes
	v1 := g.router.Group("/api/v1")
	{
		// User routes
		users := v1.Group("/users")
		{
			users.POST("", g.createUser)
			users.GET("/:id", g.getUser)
			users.GET("", g.listUsers)
			users.PUT("/:id", g.updateUser)
			users.DELETE("/:id", g.deleteUser)
		}

		// Order routes
		orders := v1.Group("/orders")
		{
			orders.POST("", g.createOrder)
			orders.GET("/:id", g.getOrder)
			orders.GET("", g.listOrders)
			orders.PUT("/:id/status", g.updateOrderStatus)
		}
	}

	// Swagger
	g.router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
}

func (g *Gateway) Start() error {
	addr := g.config.Gateway.Host + ":" + strconv.Itoa(g.config.Gateway.Port)
	g.logger.Info("Gateway starting", zap.String("address", addr))
	return g.router.Run(addr)
}

// Close closes the gateway and its connections
func (g *Gateway) Close() error {
	return g.grpcClients.Close()
}

// User handlers

func (g *Gateway) createUser(c *gin.Context) {
	var req proto.CreateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := g.grpcClients.UserClient().CreateUser(ctx, &req)
	if err != nil {
		g.logger.Error("Failed to create user", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if resp.Error != "" {
		c.JSON(http.StatusInternalServerError, gin.H{"error": resp.Error})
		return
	}

	c.JSON(http.StatusCreated, resp.User)
}

func (g *Gateway) getUser(c *gin.Context) {
	id := c.Param("id")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := g.grpcClients.UserClient().GetUser(ctx, &proto.GetUserRequest{Id: id})
	if err != nil {
		g.logger.Error("Failed to get user", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if resp.Error != "" {
		c.JSON(http.StatusNotFound, gin.H{"error": resp.Error})
		return
	}

	c.JSON(http.StatusOK, resp.User)
}

func (g *Gateway) listUsers(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := g.grpcClients.UserClient().ListUsers(ctx, &proto.ListUsersRequest{
		Page:     int32(page),
		PageSize: int32(pageSize),
	})
	if err != nil {
		g.logger.Error("Failed to list users", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if resp.Error != "" {
		c.JSON(http.StatusInternalServerError, gin.H{"error": resp.Error})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"users": resp.Users,
		"total": resp.Total,
	})
}

func (g *Gateway) updateUser(c *gin.Context) {
	id := c.Param("id")
	var req proto.UpdateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	req.Id = id

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := g.grpcClients.UserClient().UpdateUser(ctx, &req)
	if err != nil {
		g.logger.Error("Failed to update user", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if resp.Error != "" {
		c.JSON(http.StatusInternalServerError, gin.H{"error": resp.Error})
		return
	}

	c.JSON(http.StatusOK, resp.User)
}

func (g *Gateway) deleteUser(c *gin.Context) {
	id := c.Param("id")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := g.grpcClients.UserClient().DeleteUser(ctx, &proto.DeleteUserRequest{Id: id})
	if err != nil {
		g.logger.Error("Failed to delete user", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if resp.Error != "" {
		c.JSON(http.StatusInternalServerError, gin.H{"error": resp.Error})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": resp.Success})
}

// Order handlers

func (g *Gateway) createOrder(c *gin.Context) {
	var req proto.CreateOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := g.grpcClients.OrderClient().CreateOrder(ctx, &req)
	if err != nil {
		g.logger.Error("Failed to create order", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if resp.Error != "" {
		c.JSON(http.StatusInternalServerError, gin.H{"error": resp.Error})
		return
	}

	c.JSON(http.StatusCreated, resp.Order)
}

func (g *Gateway) getOrder(c *gin.Context) {
	id := c.Param("id")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := g.grpcClients.OrderClient().GetOrder(ctx, &proto.GetOrderRequest{Id: id})
	if err != nil {
		g.logger.Error("Failed to get order", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if resp.Error != "" {
		c.JSON(http.StatusNotFound, gin.H{"error": resp.Error})
		return
	}

	c.JSON(http.StatusOK, resp.Order)
}

func (g *Gateway) listOrders(c *gin.Context) {
	userID := c.Query("user_id")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := g.grpcClients.OrderClient().ListOrders(ctx, &proto.ListOrdersRequest{
		UserId:   userID,
		Page:     int32(page),
		PageSize: int32(pageSize),
	})
	if err != nil {
		g.logger.Error("Failed to list orders", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if resp.Error != "" {
		c.JSON(http.StatusInternalServerError, gin.H{"error": resp.Error})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"orders": resp.Orders,
		"total":  resp.Total,
	})
}

func (g *Gateway) updateOrderStatus(c *gin.Context) {
	id := c.Param("id")
	var req struct {
		Status string `json:"status"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := g.grpcClients.OrderClient().UpdateOrderStatus(ctx, &proto.UpdateOrderStatusRequest{
		OrderId: id,
		Status:  req.Status,
	})
	if err != nil {
		g.logger.Error("Failed to update order status", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if resp.Error != "" {
		c.JSON(http.StatusInternalServerError, gin.H{"error": resp.Error})
		return
	}

	c.JSON(http.StatusOK, resp.Order)
}

func loggerMiddleware(logger *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		query := c.Request.URL.RawQuery

		c.Next()

		logger.Info("HTTP request",
			zap.String("method", c.Request.Method),
			zap.String("path", path),
			zap.String("query", query),
			zap.Int("status", c.Writer.Status()),
			zap.Duration("latency", time.Since(start)),
		)
	}
}
