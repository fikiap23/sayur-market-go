package repository

import (
	"context"
	"product-service/internal/core/domain/model"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type OutboxRepositoryInterface interface {
	// Insert creates an outbox event using the given DB handle (transaction-safe).
	Insert(tx *gorm.DB, event *model.OutboxEvent) error

	// FetchPending returns up to `limit` PENDING events ordered by created_at ASC.
	// Uses SELECT ... FOR UPDATE SKIP LOCKED to allow safe concurrent polling.
	FetchPending(ctx context.Context, limit int) ([]model.OutboxEvent, error)

	// MarkProcessed sets status=PROCESSED and records processed_at.
	MarkProcessed(ctx context.Context, id string) error

	// IncrementRetry bumps retry_count by 1.
	IncrementRetry(ctx context.Context, id string) error

	// MarkFailed sets status=FAILED for events that exceeded max retries.
	MarkFailed(ctx context.Context, id string) error

	// CleanupProcessed deletes PROCESSED events older than the given duration.
	CleanupProcessed(ctx context.Context, olderThan time.Duration) (int64, error)
}

type outboxRepository struct {
	db *gorm.DB
}

func NewOutboxRepository(db *gorm.DB) OutboxRepositoryInterface {
	return &outboxRepository{db: db}
}

// workerDB returns a handle that skips SQL logs (avoids noisy polling under db.Debug()).
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

	// FOR UPDATE SKIP LOCKED prevents multiple worker instances from picking up
	// the same rows, which is critical for horizontal scaling.
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
