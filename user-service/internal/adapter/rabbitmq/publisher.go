package rabbitmq

import (
	"context"
	"fmt"
	"math"
	"sync"
	"sync/atomic"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/rs/zerolog"
)

const (
	defaultPublisherPoolSize = 10
	defaultPublishTimeout    = 5 * time.Second
	defaultPublishMaxRetries = 3
	defaultPublishBaseDelay  = 100 * time.Millisecond
	defaultPublishMaxDelay   = 5 * time.Second
)

// PublisherOption configures the Publisher.
type PublisherOption func(*Publisher)

// Publisher sends messages to RabbitMQ with channel pooling, publisher confirms,
// and retry with exponential backoff. Channels are kept in confirm mode so every
// successful Publish guarantees the broker has accepted the message.
type Publisher struct {
	connMgr *ConnectionManager

	pool     chan *amqp.Channel
	poolSize int

	maxRetries     int
	publishTimeout time.Duration
	baseDelay      time.Duration
	maxDelay       time.Duration

	declaredQueues   map[string]struct{}
	declaredQueuesMu sync.RWMutex

	metrics *Metrics
	logger  zerolog.Logger
	closed  int32
}

func WithPoolSize(size int) PublisherOption {
	return func(p *Publisher) {
		if size > 0 {
			p.poolSize = size
		}
	}
}

func WithPublishTimeout(d time.Duration) PublisherOption {
	return func(p *Publisher) {
		if d > 0 {
			p.publishTimeout = d
		}
	}
}

func WithPublishMaxRetries(n int) PublisherOption {
	return func(p *Publisher) {
		if n >= 0 {
			p.maxRetries = n
		}
	}
}

func WithPublisherMetrics(m *Metrics) PublisherOption {
	return func(p *Publisher) {
		p.metrics = m
	}
}

// NewPublisher creates a publisher with a pre-warmed channel pool. Returns an
// error if the initial connection to RabbitMQ fails (fail-fast at startup).
func NewPublisher(connMgr *ConnectionManager, logger zerolog.Logger, opts ...PublisherOption) (*Publisher, error) {
	p := &Publisher{
		connMgr:        connMgr,
		poolSize:       defaultPublisherPoolSize,
		maxRetries:     defaultPublishMaxRetries,
		publishTimeout: defaultPublishTimeout,
		baseDelay:      defaultPublishBaseDelay,
		maxDelay:       defaultPublishMaxDelay,
		declaredQueues: make(map[string]struct{}),
		logger:         logger.With().Str("component", "publisher").Logger(),
	}
	for _, opt := range opts {
		opt(p)
	}

	p.pool = make(chan *amqp.Channel, p.poolSize)
	for i := 0; i < p.poolSize; i++ {
		ch, err := p.createConfirmChannel()
		if err != nil {
			p.drainPool()
			return nil, fmt.Errorf("init publisher channel #%d: %w", i+1, err)
		}
		p.pool <- ch
	}

	p.logger.Info().Int("pool_size", p.poolSize).Msg("publisher initialized")
	return p, nil
}

func (p *Publisher) createConfirmChannel() (*amqp.Channel, error) {
	ch, _, err := p.connMgr.OpenChannel()
	if err != nil {
		return nil, err
	}
	if err := ch.Confirm(false); err != nil {
		_ = ch.Close()
		return nil, fmt.Errorf("enable confirm mode: %w", err)
	}
	return ch, nil
}

func (p *Publisher) borrowChannel() (*amqp.Channel, error) {
	select {
	case ch := <-p.pool:
		if ch != nil {
			return ch, nil
		}
	default:
	}
	return p.createConfirmChannel()
}

func (p *Publisher) returnChannel(ch *amqp.Channel) {
	if ch == nil || atomic.LoadInt32(&p.closed) == 1 {
		if ch != nil {
			_ = ch.Close()
		}
		return
	}
	select {
	case p.pool <- ch:
	default:
		_ = ch.Close()
	}
}

// SetupQueue declares a durable queue. Call at startup for queues this
// publisher will write to. Not required if the consumer service creates
// the queue first.
func (p *Publisher) SetupQueue(name string, args amqp.Table) error {
	ch, err := p.borrowChannel()
	if err != nil {
		return fmt.Errorf("borrow channel for queue setup: %w", err)
	}
	defer p.returnChannel(ch)

	if _, err := ch.QueueDeclare(name, true, false, false, false, args); err != nil {
		return fmt.Errorf("declare queue %s: %w", name, err)
	}

	p.declaredQueuesMu.Lock()
	p.declaredQueues[name] = struct{}{}
	p.declaredQueuesMu.Unlock()

	p.logger.Info().Str("queue", name).Msg("queue setup complete")
	return nil
}

// Publish sends a JSON message to the specified queue with publisher confirms
// and automatic retry with exponential backoff. Default behaviour is guaranteed
// delivery; use WithFireAndForget() to skip the confirm wait.
func (p *Publisher) Publish(ctx context.Context, queue string, body []byte, opts ...PublishMsgOption) error {
	cfg := defaultPublishMsgConfig()
	for _, o := range opts {
		o(cfg)
	}

	var lastErr error
	for attempt := 0; attempt <= p.maxRetries; attempt++ {
		if attempt > 0 {
			delay := p.backoff(attempt)
			p.logger.Debug().
				Int("attempt", attempt).
				Dur("backoff", delay).
				Str("queue", queue).
				Msg("retrying publish")

			if p.metrics != nil {
				p.metrics.IncrRetryCount()
			}

			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(delay):
			}
		}

		if err := p.publishOnce(ctx, queue, body, cfg); err != nil {
			lastErr = err
			if p.metrics != nil {
				p.metrics.IncrPublishErrorCount()
			}
			p.logger.Warn().Err(err).
				Int("attempt", attempt+1).
				Str("queue", queue).
				Int("size", len(body)).
				Msg("publish attempt failed")
			continue
		}

		if p.metrics != nil {
			p.metrics.IncrPublishCount()
		}
		return nil
	}

	return fmt.Errorf("publish to %s failed after %d attempts: %w", queue, p.maxRetries+1, lastErr)
}

func (p *Publisher) publishOnce(ctx context.Context, queue string, body []byte, cfg *publishMsgConfig) error {
	ch, err := p.borrowChannel()
	if err != nil {
		return fmt.Errorf("borrow channel: %w", err)
	}

	pubCtx, cancel := context.WithTimeout(ctx, p.publishTimeout)
	defer cancel()

	msg := amqp.Publishing{
		ContentType:  cfg.contentType,
		DeliveryMode: amqp.Persistent,
		Body:         body,
		Headers:      cfg.headers,
	}

	if cfg.fireAndForget {
		if err := ch.PublishWithContext(pubCtx, cfg.exchange, queue, false, false, msg); err != nil {
			_ = ch.Close()
			return err
		}
		p.returnChannel(ch)
		return nil
	}

	conf, err := ch.PublishWithDeferredConfirmWithContext(pubCtx, cfg.exchange, queue, false, false, msg)
	if err != nil {
		_ = ch.Close()
		return fmt.Errorf("publish: %w", err)
	}

	acked, err := conf.WaitContext(pubCtx)
	if err != nil {
		_ = ch.Close()
		return fmt.Errorf("confirm wait: %w", err)
	}
	if !acked {
		p.returnChannel(ch)
		return fmt.Errorf("message nacked by broker")
	}

	p.returnChannel(ch)
	return nil
}

func (p *Publisher) backoff(attempt int) time.Duration {
	d := time.Duration(float64(p.baseDelay) * math.Pow(2, float64(attempt-1)))
	if d > p.maxDelay {
		d = p.maxDelay
	}
	return d
}

// Close drains the channel pool and marks the publisher as shut down.
func (p *Publisher) Close() error {
	if !atomic.CompareAndSwapInt32(&p.closed, 0, 1) {
		return nil
	}
	p.drainPool()
	p.logger.Info().Msg("publisher closed")
	return nil
}

func (p *Publisher) drainPool() {
	for {
		select {
		case ch := <-p.pool:
			if ch != nil {
				_ = ch.Close()
			}
		default:
			return
		}
	}
}

// --- Per-message publish options ---

type publishMsgConfig struct {
	exchange      string
	contentType   string
	headers       amqp.Table
	fireAndForget bool
}

func defaultPublishMsgConfig() *publishMsgConfig {
	return &publishMsgConfig{contentType: "application/json"}
}

// PublishMsgOption configures a single Publish call.
type PublishMsgOption func(*publishMsgConfig)

// WithFireAndForget skips the publisher-confirm wait. The message may be
// lost if the broker rejects it.
func WithFireAndForget() PublishMsgOption {
	return func(c *publishMsgConfig) { c.fireAndForget = true }
}

func WithMsgHeaders(h amqp.Table) PublishMsgOption {
	return func(c *publishMsgConfig) { c.headers = h }
}

func WithMsgContentType(ct string) PublishMsgOption {
	return func(c *publishMsgConfig) { c.contentType = ct }
}

func WithExchange(ex string) PublishMsgOption {
	return func(c *publishMsgConfig) { c.exchange = ex }
}
