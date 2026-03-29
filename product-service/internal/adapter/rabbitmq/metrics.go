package rabbitmq

import "sync/atomic"

// Metrics provides atomic counters for RabbitMQ publish/consume observability.
// Pass the same instance to both Publisher and Consumer to get a unified view.
// All methods are nil-safe (callers don't need to nil-check).
type Metrics struct {
	publishCount      int64
	publishErrorCount int64
	consumeCount      int64
	ackCount          int64
	nackCount         int64
	retryCount        int64
}

func NewMetrics() *Metrics { return &Metrics{} }

func (m *Metrics) IncrPublishCount()      { atomic.AddInt64(&m.publishCount, 1) }
func (m *Metrics) IncrPublishErrorCount() { atomic.AddInt64(&m.publishErrorCount, 1) }
func (m *Metrics) IncrConsumeCount()      { atomic.AddInt64(&m.consumeCount, 1) }
func (m *Metrics) IncrAckCount()          { atomic.AddInt64(&m.ackCount, 1) }
func (m *Metrics) IncrNackCount()         { atomic.AddInt64(&m.nackCount, 1) }
func (m *Metrics) IncrRetryCount()        { atomic.AddInt64(&m.retryCount, 1) }

// MetricsSnapshot is a point-in-time copy of all counters.
type MetricsSnapshot struct {
	PublishCount      int64 `json:"publish_count"`
	PublishErrorCount int64 `json:"publish_error_count"`
	ConsumeCount      int64 `json:"consume_count"`
	AckCount          int64 `json:"ack_count"`
	NackCount         int64 `json:"nack_count"`
	RetryCount        int64 `json:"retry_count"`
}

func (m *Metrics) Snapshot() MetricsSnapshot {
	return MetricsSnapshot{
		PublishCount:      atomic.LoadInt64(&m.publishCount),
		PublishErrorCount: atomic.LoadInt64(&m.publishErrorCount),
		ConsumeCount:      atomic.LoadInt64(&m.consumeCount),
		AckCount:          atomic.LoadInt64(&m.ackCount),
		NackCount:         atomic.LoadInt64(&m.nackCount),
		RetryCount:        atomic.LoadInt64(&m.retryCount),
	}
}
