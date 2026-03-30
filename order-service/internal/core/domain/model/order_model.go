package model

import (
	"time"

	"gorm.io/gorm"
)

type Order struct {
	ID           int64          `gorm:"primaryKey"`
	OrderCode    string         `gorm:"column:order_code;unique;not null;size:64"`
	BuyerId      int64          `gorm:"column:buyer_id;not null"` // Assuming buyer_id is a user ID
	OrderDate    time.Time      `gorm:"column:order_date;not null;default:CURRENT_TIMESTAMP"`
	Status       string         `gorm:"column:status;not null;default:'pending';size:20"`
	TotalAmount  float64        `gorm:"column:total_amount;not null;default:0"`
	PaymentType  string         `gorm:"column:payment_type;not null;default:'';size:32"`
	ShippingType string         `gorm:"column:shipping_type;not null;default:'PICKUP';size:20"`
	ShippingFee  float64        `gorm:"column:shipping_fee;not null;default:0"`
	// Nullable TIME in PostgreSQL; pointer distinguishes NULL from empty string.
	OrderTime    *string        `gorm:"column:order_time"`
	Remarks      string         `gorm:"column:remarks"`
	CreatedAt    time.Time      `gorm:"column:created_at;default:CURRENT_TIMESTAMP"`
	UpdatedAt    *time.Time     `gorm:"column:updated_at"`
	DeletedAt    gorm.DeletedAt `gorm:"column:deleted_at;index"`
	OrderItems   []OrderItem    `gorm:"foreignKey:OrderID"`
}
