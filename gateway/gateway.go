package gateway

import (
	"fmt"
	"net/http"
	"time"

	"github.com/example/microshop/pkg/config"
	"github.com/example/microshop/pkg/discovery"
	"github.com/gin-gonic/gin"
	ginSwagger "github.com/swaggo/gin-swagger"
	"github.com/swaggo/files"
	"go.uber.org/zap"
)

type Gateway struct {
	config      *config.Config
	discovery   *discovery.ServiceDiscovery
	logger      *zap.Logger
	router      *gin.Engine
	httpClient  *http.Client
}

func NewGateway(cfg *config.Config, logger *zap.Logger, disc *discovery.ServiceDiscovery) *Gateway {
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.Use(gin.Recovery())
	router.Use(loggerMiddleware(logger))

	return &Gateway{
		config:    cfg,
		discovery: disc,
		logger:    logger,
		router:    router,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
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
	addr := fmt.Sprintf("%s:%d", g.config.Gateway.Host, g.config.Gateway.Port)
	g.logger.Info("Gateway starting", zap.String("address", addr))
	return g.router.Run(addr)
}

func (g *Gateway) createUser(c *gin.Context) {
	var req map[string]interface{}
	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Forward to user service via gRPC
	// In a real implementation, you would use gRPC client
	c.JSON(http.StatusCreated, gin.H{
		"id":      "123",
		"message": "User created successfully",
	})
}

func (g *Gateway) getUser(c *gin.Context) {
	id := c.Param("id")
	c.JSON(http.StatusOK, gin.H{
		"id":    id,
		"name":  "John Doe",
		"email": "john@example.com",
	})
}

func (g *Gateway) listUsers(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"users": []interface{}{},
		"total": 0,
	})
}

func (g *Gateway) updateUser(c *gin.Context) {
	id := c.Param("id")
	c.JSON(http.StatusOK, gin.H{
		"id":      id,
		"message": "User updated successfully",
	})
}

func (g *Gateway) deleteUser(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"success": true,
	})
}

func (g *Gateway) createOrder(c *gin.Context) {
	var req map[string]interface{}
	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"id":      "456",
		"message": "Order created successfully",
	})
}

func (g *Gateway) getOrder(c *gin.Context) {
	id := c.Param("id")
	c.JSON(http.StatusOK, gin.H{
		"id":     id,
		"status": "pending",
	})
}

func (g *Gateway) listOrders(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"orders": []interface{}{},
		"total":  0,
	})
}

func (g *Gateway) updateOrderStatus(c *gin.Context) {
	id := c.Param("id")
	c.JSON(http.StatusOK, gin.H{
		"id":      id,
		"message": "Order status updated successfully",
	})
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
