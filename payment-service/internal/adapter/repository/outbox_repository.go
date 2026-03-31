package repository

import (
	"context"
	"time"

	"payment-service/internal/core/domain/model"

	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type OutboxRepositoryInterface interface {
	Insert(tx *gorm.DB, event *model.OutboxEvent) error
	FetchPending(ctx context.Context, limit int) ([]model.OutboxEvent, error)
	MarkProcessed(ctx context.Context, id string) error
	IncrementRetry(ctx context.Context, id string) error
	MarkFailed(ctx context.Context, id string) error
	CleanupProcessed(ctx context.Context, olderThan time.Duration) (int64, error)
}

type outboxRepository struct {
	db *gorm.DB
}

func NewOutboxRepository(db *gorm.DB) OutboxRepositoryInterface {
	return &outboxRepository{db: db}
}

func (r *outboxRepository) workerDB(ctx context.Context) *gorm.DB {
	return r.db.WithContext(ctx).Session(&gorm.Session{
		Logger: logger.Default.LogMode(logger.Silent),
	})
}

func (r *outboxRepository) Insert(tx *gorm.DB, event *model.OutboxEvent) error {
	return tx.Create(event).Error
}

func (r *outboxRepository) FetchPending(ctx context.Context, limit int) ([]model.OutboxEvent, error) {
	var events []model.OutboxEvent
	err := r.workerDB(ctx).
		Where("status = ?", model.OutboxStatusPending).
		Order("created_at ASC").
		Limit(limit).
		Set("gorm:query_option", "FOR UPDATE SKIP LOCKED").
		Find(&events).Error
	return events, err
}

func (r *outboxRepository) MarkProcessed(ctx context.Context, id string) error {
	now := time.Now()
	return r.workerDB(ctx).
		Model(&model.OutboxEvent{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"status":       model.OutboxStatusProcessed,
			"processed_at": now,
		}).Error
}

func (r *outboxRepository) IncrementRetry(ctx context.Context, id string) error {
	return r.workerDB(ctx).
		Model(&model.OutboxEvent{}).
		Where("id = ?", id).
		Update("retry_count", gorm.Expr("retry_count + 1")).Error
}

func (r *outboxRepository) MarkFailed(ctx context.Context, id string) error {
	return r.workerDB(ctx).
		Model(&model.OutboxEvent{}).
		Where("id = ?", id).
		Update("status", model.OutboxStatusFailed).Error
}

func (r *outboxRepository) CleanupProcessed(ctx context.Context, olderThan time.Duration) (int64, error) {
	cutoff := time.Now().Add(-olderThan)
	result := r.workerDB(ctx).
		Where("status = ? AND processed_at < ?", model.OutboxStatusProcessed, cutoff).
		Delete(&model.OutboxEvent{})
	return result.RowsAffected, result.Error
}
