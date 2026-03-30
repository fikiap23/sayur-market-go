package model

import "time"

// StockDeduction records an applied stock deduction for idempotent RabbitMQ consumption.
// Rows are keyed by dedup_key (unique); see entity.DedupKeyForStock.
type StockDeduction struct {
	ID        int64     `gorm:"primaryKey;autoIncrement"`
	DedupKey  string    `gorm:"column:dedup_key;uniqueIndex;size:512;not null"`
	ProductID int64     `gorm:"column:product_id;not null;index"`
	Quantity  int64     `gorm:"column:quantity;not null"`
	CreatedAt time.Time `gorm:"column:created_at;autoCreateTime"`
}

func (StockDeduction) TableName() string {
	return "stock_deductions"
}
