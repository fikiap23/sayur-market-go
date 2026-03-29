package rabbitmq

import (
	"fmt"
	"sync"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/rs/zerolog"
)

// ConnectionManager holds a single shared AMQP connection reused by all
// publishers and consumers within a service. Each caller obtains its own
// channel via OpenChannel. Reconnection is handled transparently under a
// mutex so concurrent goroutines never race on dial.
type ConnectionManager struct {
	url    string
	mu     sync.Mutex
	conn   *amqp.Connection
	logger zerolog.Logger
}

func NewConnectionManager(amqpURL string, logger zerolog.Logger) *ConnectionManager {
	return &ConnectionManager{
		url:    amqpURL,
		logger: logger.With().Str("component", "rabbitmq_conn").Logger(),
	}
}

func (cm *ConnectionManager) dial() error {
	conn, err := amqp.Dial(cm.url)
	if err != nil {
		return fmt.Errorf("dial rabbitmq: %w", err)
	}
	cm.conn = conn
	cm.logger.Info().Msg("connection established")
	return nil
}

// OpenChannel returns a new AMQP channel together with the underlying
// connection (useful for NotifyClose). If the shared connection is nil
// or closed the manager reconnects first.
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
		if cm.conn != nil && !cm.conn.IsClosed() {
			_ = cm.conn.Close()
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
		cm.logger.Info().Msg("closing connection")
		return cm.conn.Close()
	}
	return nil
}
