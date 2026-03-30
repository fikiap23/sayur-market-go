package model

import (
	"time"

	"gorm.io/gorm"
)

const (
	OutboxStatusPending   = "PENDING"
	OutboxStatusProcessed = "PROCESSED"
	OutboxStatusFailed    = "FAILED"

	EventOrderCreated           = "order.created"
	EventOrderDeleted           = "order.deleted"
	EventOrderStockUpdate       = "order.stock.update"
	EventOrderStatusES          = "order.status.elasticsearch"
	EventNotificationOrderEmail = "notification.order.email"
	EventNotificationOrderPush  = "notification.order.push"
)

type OutboxEvent struct {
	ID          string         `gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	EventType   string         `gorm:"column:event_type;not null;size:50"`
	Payload     string         `gorm:"column:payload;type:jsonb;not null"`
	Status      string         `gorm:"column:status;not null;size:20;default:PENDING"`
	RetryCount  int            `gorm:"column:retry_count;not null;default:0"`
	CreatedAt   time.Time      `gorm:"column:created_at;autoCreateTime"`
	ProcessedAt *time.Time     `gorm:"column:processed_at"`
	DeletedAt   gorm.DeletedAt `gorm:"column:deleted_at;index"`
}

func (OutboxEvent) TableName() string {
	return "outbox_events"
}
