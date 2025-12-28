package main

import (
	"fmt"
	"time"

	"github.com/asynkron/protoactor-go/actor"
	"github.com/example/microshop/pkg/config"
	"go.uber.org/zap"
)

// OrderActor handles order-related messages
type OrderActor struct {
	logger *zap.Logger
}

func (a *OrderActor) Receive(ctx actor.Context) {
	switch msg := ctx.Message().(type) {
	case *CreateOrder:
		a.logger.Info("Creating order",
			zap.String("user_id", msg.UserID),
			zap.Int32("item_count", int32(len(msg.Items))))

		// Simulate order processing
		time.Sleep(100 * time.Millisecond)

		ctx.Respond(&OrderResponse{
			OrderID:  fmt.Sprintf("ORD-%d", time.Now().UnixNano()),
			Status:   "created",
			Message:  "Order created successfully",
		})

	case *GetOrderStatus:
		a.logger.Info("Getting order status", zap.String("order_id", msg.OrderID))

		ctx.Respond(&OrderStatus{
			OrderID: msg.OrderID,
			Status:  "processing",
		})

	case *actor.Started:
		a.logger.Info("Order actor started")

	case *actor.Stopping:
		a.logger.Info("Order actor stopping")

	case *actor.Stopped:
		a.logger.Info("Order actor stopped")
	}
}

// Messages
type CreateOrder struct {
	UserID string
	Items  []OrderItem
}

type OrderItem struct {
	ProductID   string
	ProductName string
	Quantity    int32
	Price       float64
}

type OrderResponse struct {
	OrderID string
	Status  string
	Message string
}

type GetOrderStatus struct {
	OrderID string
}

type OrderStatus struct {
	OrderID string
	Status  string
}

// NotificationActor handles notifications
type NotificationActor struct {
	logger *zap.Logger
}

func (a *NotificationActor) Receive(ctx actor.Context) {
	switch msg := ctx.Message().(type) {
	case *SendNotification:
		a.logger.Info("Sending notification",
			zap.String("recipient", msg.Recipient),
			zap.String("type", msg.Type),
			zap.String("message", msg.Message))

		// Simulate sending notification
		ctx.Respond(&NotificationResponse{
			Success: true,
			Message: "Notification sent successfully",
		})

	case *actor.Started:
		a.logger.Info("Notification actor started")
	}
}

type SendNotification struct {
	Recipient string
	Type      string // email, sms, push
	Message   string
}

type NotificationResponse struct {
	Success bool
	Message string
}

// OrderClusterActor - clustered order actor
type OrderClusterActor struct {
	orders map[string]*OrderInfo
}

func (a *OrderClusterActor) Receive(ctx actor.Context) {
	switch msg := ctx.Message().(type) {
	case *actor.Started:
		a.orders = make(map[string]*OrderInfo)

	case *CreateOrderCluster:
		orderID := fmt.Sprintf("ORD-%d", time.Now().UnixNano())
		a.orders[orderID] = &OrderInfo{
			OrderID:     orderID,
			UserID:      msg.UserID,
			Items:       msg.Items,
			Status:      "pending",
			CreatedAt:   time.Now(),
		}
		ctx.Respond(&OrderResponse{OrderID: orderID, Status: "pending"})

	case *GetOrderStatusCluster:
		if order, ok := a.orders[msg.OrderID]; ok {
			ctx.Respond(&OrderStatus{OrderID: order.OrderID, Status: order.Status})
		} else {
			ctx.Respond(&OrderStatus{OrderID: msg.OrderID, Status: "not found"})
		}
	}
}

type CreateOrderCluster struct {
	UserID string
	Items  []OrderItem
}

type GetOrderStatusCluster struct {
	OrderID string
}

type OrderInfo struct {
	OrderID   string
	UserID    string
	Items     []OrderItem
	Status    string
	CreatedAt time.Time
}

// StartActorService starts the ProtoActor service
func StartActorService(cfg *config.Config, logger *zap.Logger) error {
	// Create actor system
	system := actor.NewActorSystem()

	// Start local order actor
	orderProps := actor.PropsFromProducer(func() actor.Actor {
		return &OrderActor{logger: logger.Named("order-actor")}
	})
	orderPid, err := system.Root.SpawnNamed(orderProps, "order-actor")
	if err != nil {
		return fmt.Errorf("failed to spawn order actor: %w", err)
	}

	// Start notification actor
	notificationProps := actor.PropsFromProducer(func() actor.Actor {
		return &NotificationActor{logger: logger.Named("notification-actor")}
	})
	_, err = system.Root.SpawnNamed(notificationProps, "notification-actor")
	if err != nil {
		return fmt.Errorf("failed to spawn notification actor: %w", err)
	}

	logger.Info("Local actors started",
		zap.String("order_actor", orderPid.Id))

	// Example: Send a message to order actor
	go func() {
		time.Sleep(2 * time.Second)
		future := system.Root.RequestFuture(orderPid, &CreateOrder{
			UserID: "user-123",
			Items: []OrderItem{
				{ProductID: "prod-1", ProductName: "Product 1", Quantity: 2, Price: 99.99},
			},
		}, 5*time.Second)

		result, err := future.Result()
		if err != nil {
			logger.Error("Failed to get response", zap.Error(err))
			return
		}

		if resp, ok := result.(*OrderResponse); ok {
			logger.Info("Order created",
				zap.String("order_id", resp.OrderID),
				zap.String("status", resp.Status))
		}
	}()

	return nil
}
