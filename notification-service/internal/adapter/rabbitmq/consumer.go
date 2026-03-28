package rabbitmq

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/rs/zerolog"
)

const (
	defaultWorkerPoolSize = 10
	defaultPrefetchCount  = 10
	defaultMaxRetries     = 3
	defaultProcessTimeout = 30 * time.Second
	defaultReconnectDelay = 5 * time.Second

	retryCountHeader = "x-retry-count"
)

// MessageHandler defines the contract for processing consumed messages.
// Implementations must be safe for concurrent use.
type MessageHandler interface {
	Handle(ctx context.Context, body []byte) error
}

// ConsumerOption applies optional configuration to a Consumer.
type ConsumerOption func(*Consumer)

// Consumer manages a RabbitMQ queue subscription with a bounded worker pool,
// manual ACK/NACK, retry with republish, optional DLQ, and automatic reconnection.
// It obtains channels from a shared ConnectionManager rather than owning a
// dedicated AMQP connection.
type Consumer struct {
	connMgr        *ConnectionManager
	channel        *amqp.Channel
	handler        MessageHandler
	queueName      string
	workerPoolSize int
	prefetchCount  int
	maxRetries     int
	processTimeout time.Duration
	reconnectDelay time.Duration
	enableDLQ      bool
	logger         zerolog.Logger
}

func WithWorkerPoolSize(size int) ConsumerOption {
	return func(c *Consumer) {
		if size > 0 {
			c.workerPoolSize = size
		}
	}
}

func WithPrefetchCount(count int) ConsumerOption {
	return func(c *Consumer) {
		if count > 0 {
			c.prefetchCount = count
		}
	}
}

func WithMaxRetries(n int) ConsumerOption {
	return func(c *Consumer) {
		if n >= 0 {
			c.maxRetries = n
		}
	}
}

func WithProcessTimeout(d time.Duration) ConsumerOption {
	return func(c *Consumer) {
		if d > 0 {
			c.processTimeout = d
		}
	}
}

func WithReconnectDelay(d time.Duration) ConsumerOption {
	return func(c *Consumer) {
		if d > 0 {
			c.reconnectDelay = d
		}
	}
}

func WithDLQ(enable bool) ConsumerOption {
	return func(c *Consumer) {
		c.enableDLQ = enable
	}
}

func NewConsumer(
	connMgr *ConnectionManager,
	queueName string,
	handler MessageHandler,
	logger zerolog.Logger,
	opts ...ConsumerOption,
) *Consumer {
	c := &Consumer{
		connMgr:        connMgr,
		handler:        handler,
		queueName:      queueName,
		workerPoolSize: defaultWorkerPoolSize,
		prefetchCount:  defaultPrefetchCount,
		maxRetries:     defaultMaxRetries,
		processTimeout: defaultProcessTimeout,
		reconnectDelay: defaultReconnectDelay,
		logger:         logger.With().Str("component", "consumer").Str("queue", queueName).Logger(),
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

// Start begins consuming messages and blocks until ctx is cancelled.
// On channel / connection loss it re-opens a channel automatically.
func (c *Consumer) Start(ctx context.Context) error {
	for {
		err := c.run(ctx)
		if ctx.Err() != nil {
			c.cleanup()
			c.logger.Info().Msg("consumer stopped")
			return ctx.Err()
		}

		c.logger.Warn().Err(err).Msg("consumer disconnected, reconnecting...")
		c.cleanup()

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(c.reconnectDelay):
		}
	}
}

// run performs a single channel session. It returns when the channel or
// connection drops, or ctx is cancelled, allowing Start to retry.
func (c *Consumer) run(ctx context.Context) error {
	ch, conn, err := c.connMgr.OpenChannel()
	if err != nil {
		return fmt.Errorf("open channel: %w", err)
	}
	c.channel = ch

	if err := c.setupQueue(); err != nil {
		return fmt.Errorf("setup queue: %w", err)
	}

	if err := c.channel.Qos(c.prefetchCount, 0, false); err != nil {
		return fmt.Errorf("set QoS: %w", err)
	}

	deliveries, err := c.channel.Consume(
		c.queueName,
		"",    // auto-generated consumer tag
		false, // autoAck disabled — we ACK/NACK manually
		false, // exclusive
		false, // noLocal
		false, // noWait
		nil,
	)
	if err != nil {
		return fmt.Errorf("start consume: %w", err)
	}

	c.logger.Info().
		Int("workers", c.workerPoolSize).
		Int("prefetch", c.prefetchCount).
		Int("max_retries", c.maxRetries).
		Dur("process_timeout", c.processTimeout).
		Bool("dlq_enabled", c.enableDLQ).
		Msg("consumer started")

	connClose := conn.NotifyClose(make(chan *amqp.Error, 1))
	chanClose := c.channel.NotifyClose(make(chan *amqp.Error, 1))

	jobs := make(chan amqp.Delivery, c.prefetchCount)

	var wg sync.WaitGroup
	for i := 0; i < c.workerPoolSize; i++ {
		wg.Add(1)
		go c.worker(ctx, i, jobs, &wg)
	}

	var runErr error
loop:
	for {
		select {
		case <-ctx.Done():
			break loop
		case amqpErr := <-connClose:
			runErr = fmt.Errorf("connection closed: %v", amqpErr)
			break loop
		case amqpErr := <-chanClose:
			runErr = fmt.Errorf("channel closed: %v", amqpErr)
			break loop
		case d, ok := <-deliveries:
			if !ok {
				runErr = errors.New("delivery channel closed")
				break loop
			}
			jobs <- d
		}
	}

	close(jobs)
	wg.Wait()

	return runErr
}

func (c *Consumer) setupQueue() error {
	var args amqp.Table

	if c.enableDLQ {
		dlqName := c.queueName + ".dlq"
		if _, err := c.channel.QueueDeclare(dlqName, true, false, false, false, nil); err != nil {
			return fmt.Errorf("declare DLQ %s: %w", dlqName, err)
		}
		args = amqp.Table{
			"x-dead-letter-exchange":    "",
			"x-dead-letter-routing-key": dlqName,
		}
	}

	if _, err := c.channel.QueueDeclare(c.queueName, true, false, false, false, args); err != nil {
		return fmt.Errorf("declare queue %s: %w", c.queueName, err)
	}
	return nil
}

// worker drains the jobs channel, processing one delivery at a time.
func (c *Consumer) worker(ctx context.Context, id int, jobs <-chan amqp.Delivery, wg *sync.WaitGroup) {
	defer wg.Done()

	log := c.logger.With().Int("worker_id", id).Logger()
	log.Debug().Msg("worker started")

	for d := range jobs {
		c.processMessage(ctx, d, log)
	}

	log.Debug().Msg("worker stopped")
}

func (c *Consumer) processMessage(ctx context.Context, d amqp.Delivery, log zerolog.Logger) {
	processCtx, cancel := context.WithTimeout(ctx, c.processTimeout)
	defer cancel()

	log.Info().Int("size", len(d.Body)).Msg("processing message")

	if err := c.handler.Handle(processCtx, d.Body); err != nil {
		c.handleFailure(processCtx, d, err, log)
		return
	}

	if ackErr := d.Ack(false); ackErr != nil {
		log.Error().Err(ackErr).Msg("failed to ACK message")
	}
}

// handleFailure implements retry-by-republish: on transient failure the message
// is re-published with an incremented x-retry-count header so it re-enters the
// queue tail (preserving ordering for other messages). Once maxRetries is
// exceeded the message is NACK'd without requeue; if DLQ is enabled the broker
// routes it to <queue>.dlq via the dead-letter exchange.
func (c *Consumer) handleFailure(ctx context.Context, d amqp.Delivery, processErr error, log zerolog.Logger) {
	retryCount := getRetryCount(d)

	if retryCount < c.maxRetries {
		log.Warn().
			Err(processErr).
			Int("retry", retryCount+1).
			Int("max_retries", c.maxRetries).
			Msg("retrying message")

		headers := cloneHeaders(d.Headers)
		headers[retryCountHeader] = int32(retryCount + 1)

		pubErr := c.channel.PublishWithContext(ctx, "", c.queueName, false, false, amqp.Publishing{
			Headers:     headers,
			ContentType: d.ContentType,
			Body:        d.Body,
		})
		if pubErr != nil {
			log.Error().Err(pubErr).Msg("failed to republish for retry, rejecting message")
			_ = d.Nack(false, false)
			return
		}

		if ackErr := d.Ack(false); ackErr != nil {
			log.Error().Err(ackErr).Msg("failed to ACK original after republish")
		}
		return
	}

	log.Error().
		Err(processErr).
		Int("retry", retryCount).
		Msg("max retries exceeded, rejecting message")

	_ = d.Nack(false, false)
}

func getRetryCount(d amqp.Delivery) int {
	if d.Headers == nil {
		return 0
	}
	count, ok := d.Headers[retryCountHeader]
	if !ok {
		return 0
	}
	switch v := count.(type) {
	case int32:
		return int(v)
	case int64:
		return int(v)
	case int:
		return v
	default:
		return 0
	}
}

func cloneHeaders(src amqp.Table) amqp.Table {
	dst := make(amqp.Table, len(src)+1)
	for k, v := range src {
		dst[k] = v
	}
	return dst
}

// cleanup closes only the consumer's own channel; the shared connection
// is managed by ConnectionManager.
func (c *Consumer) cleanup() {
	if c.channel != nil {
		_ = c.channel.Close()
		c.channel = nil
	}
}
