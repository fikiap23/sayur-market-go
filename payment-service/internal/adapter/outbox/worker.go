package outbox

import (
	"context"
	"time"

	"payment-service/internal/adapter/rabbitmq"
	"payment-service/internal/adapter/repository"
	"payment-service/internal/core/domain/model"

	"github.com/rs/zerolog"
)

const (
	defaultPollInterval = 2 * time.Second
	defaultBatchSize    = 50
	defaultMaxRetries   = 5
	defaultCleanupAge   = 7 * 24 * time.Hour
	defaultCleanupEvery = 1 * time.Hour
)

type WorkerConfig struct {
	PollInterval time.Duration
	BatchSize    int
	MaxRetries   int
	CleanupAge   time.Duration
	CleanupEvery time.Duration
}

func DefaultWorkerConfig() WorkerConfig {
	return WorkerConfig{
		PollInterval: defaultPollInterval,
		BatchSize:    defaultBatchSize,
		MaxRetries:   defaultMaxRetries,
		CleanupAge:   defaultCleanupAge,
		CleanupEvery: defaultCleanupEvery,
	}
}

// Worker polls outbox_events and publishes to RabbitMQ using the default
// exchange (routing key = queue name). EventType must match the queue name
// (e.g. PUBLISHER_PAYMENT_SUCCESS) so order-service consumers keep working
// without binding to a topic exchange.
type Worker struct {
	repo      repository.OutboxRepositoryInterface
	publisher *rabbitmq.Publisher
	cfg       WorkerConfig
	logger    zerolog.Logger
}

func NewWorker(
	repo repository.OutboxRepositoryInterface,
	publisher *rabbitmq.Publisher,
	cfg WorkerConfig,
	logger zerolog.Logger,
) *Worker {
	return &Worker{
		repo:      repo,
		publisher: publisher,
		cfg:       cfg,
		logger:    logger.With().Str("component", "outbox_worker").Logger(),
	}
}

// Start begins the poll loop. Blocks until ctx is cancelled.
func (w *Worker) Start(ctx context.Context) error {
	w.logger.Info().
		Dur("poll_interval", w.cfg.PollInterval).
		Int("batch_size", w.cfg.BatchSize).
		Int("max_retries", w.cfg.MaxRetries).
		Msg("outbox worker started")

	pollTicker := time.NewTicker(w.cfg.PollInterval)
	defer pollTicker.Stop()

	cleanupTicker := time.NewTicker(w.cfg.CleanupEvery)
	defer cleanupTicker.Stop()

	for {
		select {
		case <-ctx.Done():
			w.logger.Info().Msg("outbox worker stopped")
			return ctx.Err()

		case <-pollTicker.C:
			w.processBatch(ctx)

		case <-cleanupTicker.C:
			w.cleanup(ctx)
		}
	}
}

func (w *Worker) processBatch(ctx context.Context) {
	events, err := w.repo.FetchPending(ctx, w.cfg.BatchSize)
	if err != nil {
		w.logger.Error().Err(err).Msg("failed to fetch pending outbox events")
		return
	}

	if len(events) == 0 {
		return
	}

	w.logger.Debug().Int("count", len(events)).Msg("processing outbox batch")

	for _, event := range events {
		if ctx.Err() != nil {
			return
		}
		w.processEvent(ctx, event)
	}
}

func (w *Worker) processEvent(ctx context.Context, event model.OutboxEvent) {
	log := w.logger.With().
		Str("event_id", event.ID).
		Str("event_type", event.EventType).
		Int("retry_count", event.RetryCount).
		Logger()

	// event_type is the queue name (default exchange routing key)
	err := w.publisher.Publish(
		ctx,
		event.EventType,
		[]byte(event.Payload),
	)

	if err != nil {
		log.Error().Err(err).Msg("failed to publish outbox event")

		if err := w.repo.IncrementRetry(ctx, event.ID); err != nil {
			log.Error().Err(err).Msg("failed to increment retry count")
		}

		if event.RetryCount+1 >= w.cfg.MaxRetries {
			log.Error().Msg("max retries exceeded, marking event as FAILED")
			if err := w.repo.MarkFailed(ctx, event.ID); err != nil {
				log.Error().Err(err).Msg("failed to mark event as FAILED")
			}
		}
		return
	}

	if err := w.repo.MarkProcessed(ctx, event.ID); err != nil {
		log.Error().Err(err).Msg("published but failed to mark as PROCESSED")
		return
	}

	log.Info().Msg("outbox event published and marked PROCESSED")
}

func (w *Worker) cleanup(ctx context.Context) {
	deleted, err := w.repo.CleanupProcessed(ctx, w.cfg.CleanupAge)
	if err != nil {
		w.logger.Error().Err(err).Msg("outbox cleanup failed")
		return
	}
	if deleted > 0 {
		w.logger.Info().Int64("deleted", deleted).Dur("older_than", w.cfg.CleanupAge).Msg("outbox cleanup done")
	}
}
