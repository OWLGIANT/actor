package models

import (
	"time"
)

type Order struct {
	ID         string    `gorm:"primaryKey;type:varchar(36)" json:"id"`
	UserID     string    `gorm:"type:varchar(36);not null;index" json:"user_id"`
	Items      string    `gorm:"type:text" json:"items"` // JSON string
	TotalAmount float64  `gorm:"type:decimal(10,2)" json:"total_amount"`
	Status     string    `gorm:"type:varchar(20);default:'pending'" json:"status"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
	DeletedAt  *time.Time `gorm:"index" json:"-"`
}

func (Order) TableName() string {
	return "orders"
}

type OrderItem struct {
	ProductID   string  `json:"product_id"`
	ProductName string  `json:"product_name"`
	Quantity    int32   `json:"quantity"`
	Price       float64 `json:"price"`
}
