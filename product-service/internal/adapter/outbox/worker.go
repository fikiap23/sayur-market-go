package outbox

import (
	"context"
	"math"
	"product-service/internal/adapter/rabbitmq"
	"product-service/internal/adapter/repository"
	"product-service/internal/core/domain/model"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/rs/zerolog"
)

const (
	defaultPollInterval  = 2 * time.Second
	defaultBatchSize     = 50
	defaultMaxRetries    = 5
	defaultCleanupAge    = 7 * 24 * time.Hour // 7 days
	defaultCleanupEvery  = 1 * time.Hour

	ExchangeName = "product.events"
	ExchangeType = "topic"
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

// Worker polls the outbox table for PENDING events and publishes them to
// RabbitMQ via a topic exchange. On success it marks the event PROCESSED;
// on failure it increments retry_count and eventually marks FAILED.
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

// SetupExchange declares the topic exchange and binds queues so consumers
// can receive events by routing key.
func (w *Worker) SetupExchange(connMgr *rabbitmq.ConnectionManager, bindings map[string]string) error {
	ch, _, err := connMgr.OpenChannel()
	if err != nil {
		return err
	}
	defer ch.Close()

	if err := ch.ExchangeDeclare(
		ExchangeName,
		ExchangeType,
		true,  // durable
		false, // auto-delete
		false, // internal
		false, // no-wait
		nil,
	); err != nil {
		return err
	}

	w.logger.Info().Str("exchange", ExchangeName).Msg("exchange declared")

	// bindings maps routing_key -> queue_name
	for routingKey, queueName := range bindings {
		dlqName := queueName + ".dlq"
		if _, err := ch.QueueDeclare(dlqName, true, false, false, false, nil); err != nil {
			return err
		}
		args := amqp.Table{
			"x-dead-letter-exchange":    "",
			"x-dead-letter-routing-key": dlqName,
		}
		if _, err := ch.QueueDeclare(queueName, true, false, false, false, args); err != nil {
			return err
		}
		if err := ch.QueueBind(queueName, routingKey, ExchangeName, false, nil); err != nil {
			return err
		}
		w.logger.Info().
			Str("queue", queueName).
			Str("routing_key", routingKey).
			Str("exchange", ExchangeName).
			Msg("queue bound to exchange")
	}

	return nil
}

// Start begins the poll loop. It blocks until ctx is cancelled.
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

	// event_type doubles as the routing key (e.g. "product.created")
	err := w.publisher.Publish(
		ctx,
		event.EventType, // used as routing key
		[]byte(event.Payload),
		rabbitmq.WithExchange(ExchangeName),
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

// cleanup removes PROCESSED events older than CleanupAge.
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

// BackoffDuration returns an exponential backoff duration for the given retry count.
func BackoffDuration(retry int) time.Duration {
	d := time.Duration(float64(time.Second) * math.Pow(2, float64(retry)))
	if d > 5*time.Minute {
		d = 5 * time.Minute
	}
	return d
}
