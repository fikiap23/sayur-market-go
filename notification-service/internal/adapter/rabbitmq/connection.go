package rabbitmq

import (
	"fmt"
	"sync"

	"notification-service/config"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/rs/zerolog"
)

// ConnectionManager holds a single shared AMQP connection that is
// reused by all consumers. Each consumer opens its own channel via
// OpenChannel. Reconnection is handled transparently under a mutex
// so concurrent callers never race on dial.
type ConnectionManager struct {
	cfg    *config.Config
	mu     sync.Mutex
	conn   *amqp.Connection
	logger zerolog.Logger
}

func NewConnectionManager(cfg *config.Config, logger zerolog.Logger) *ConnectionManager {
	return &ConnectionManager{
		cfg:    cfg,
		logger: logger.With().Str("component", "connection_manager").Logger(),
	}
}

func (cm *ConnectionManager) dial() error {
	conn, err := cm.cfg.NewRabbitMQ()
	if err != nil {
		return fmt.Errorf("dial: %w", err)
	}
	cm.conn = conn
	cm.logger.Info().Msg("rabbitmq connection established")
	return nil
}

// OpenChannel returns a new AMQP channel together with the underlying
// connection reference (useful for registering NotifyClose). If the
// shared connection is nil or closed the manager reconnects first.
func (cm *ConnectionManager) OpenChannel() (*amqp.Channel, *amqp.Connection, error) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	if cm.conn == nil || cm.conn.IsClosed() {
		if err := cm.dial(); err != nil {
			return nil, nil, err
		}
	}

	ch, err := cm.conn.Channel()
	if err != nil {
		cm.logger.Warn().Err(err).Msg("channel open failed, reconnecting")
		if closeErr := cm.conn.Close(); closeErr != nil {
			cm.logger.Debug().Err(closeErr).Msg("close stale connection")
		}
		if err := cm.dial(); err != nil {
			return nil, nil, err
		}
		ch, err = cm.conn.Channel()
		if err != nil {
			return nil, nil, fmt.Errorf("open channel after reconnect: %w", err)
		}
	}

	return ch, cm.conn, nil
}

func (cm *ConnectionManager) Close() error {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	if cm.conn != nil && !cm.conn.IsClosed() {
		return cm.conn.Close()
	}
	return nil
}
