package message

import (
	"encoding/json"
	"fmt"
	"sync"
	"user-service/config"
	"user-service/utils"

	"github.com/labstack/gommon/log"
	"github.com/streadway/amqp"
)

const defaultPublisherChannelPoolSize = 24

var (
	rabbitConn        *amqp.Connection
	rabbitChannelPool chan *amqp.Channel
	rabbitStateMu     sync.RWMutex

	declaredQueues   = make(map[string]struct{})
	declaredQueuesMu sync.RWMutex
)

func ensureRabbitReady() error {
	rabbitStateMu.RLock()
	ready := rabbitConn != nil && !rabbitConn.IsClosed() && rabbitChannelPool != nil
	rabbitStateMu.RUnlock()
	if ready {
		return nil
	}

	rabbitStateMu.Lock()
	defer rabbitStateMu.Unlock()

	ready = rabbitConn != nil && !rabbitConn.IsClosed() && rabbitChannelPool != nil
	if ready {
		return nil
	}

	conn, err := config.NewConfig().NewRabbitMQ()
	if err != nil {
		return fmt.Errorf("connect to rabbitmq: %w", err)
	}

	pool := make(chan *amqp.Channel, defaultPublisherChannelPoolSize)
	for i := 0; i < defaultPublisherChannelPoolSize; i++ {
		ch, err := conn.Channel()
		if err != nil {
			_ = conn.Close()
			return fmt.Errorf("open publish channel #%d: %w", i+1, err)
		}
		pool <- ch
	}

	rabbitConn = conn
	rabbitChannelPool = pool
	resetDeclaredQueues()
	return nil
}

func borrowChannel() (*amqp.Channel, chan *amqp.Channel, error) {
	if err := ensureRabbitReady(); err != nil {
		return nil, nil, err
	}

	rabbitStateMu.RLock()
	pool := rabbitChannelPool
	rabbitStateMu.RUnlock()

	ch := <-pool
	return ch, pool, nil
}

func releaseChannel(ch *amqp.Channel, poolSnapshot chan *amqp.Channel) {
	if ch == nil {
		return
	}

	rabbitStateMu.RLock()
	currentPool := rabbitChannelPool
	rabbitStateMu.RUnlock()

	// If state changed (reconnect), old channels are discarded safely.
	if poolSnapshot == nil || poolSnapshot != currentPool {
		_ = ch.Close()
		return
	}

	select {
	case poolSnapshot <- ch:
	default:
		_ = ch.Close()
	}
}

func invalidateRabbitState() {
	rabbitStateMu.Lock()
	oldConn := rabbitConn
	oldPool := rabbitChannelPool
	rabbitConn = nil
	rabbitChannelPool = nil
	rabbitStateMu.Unlock()

	if oldConn != nil && !oldConn.IsClosed() {
		_ = oldConn.Close()
	}

	if oldPool != nil {
		for {
			select {
			case ch := <-oldPool:
				if ch != nil {
					_ = ch.Close()
				}
			default:
				resetDeclaredQueues()
				return
			}
		}
	}

	resetDeclaredQueues()
}

func resetDeclaredQueues() {
	declaredQueuesMu.Lock()
	declaredQueues = make(map[string]struct{})
	declaredQueuesMu.Unlock()
}

func ensureQueueDeclared(ch *amqp.Channel, queueName string) error {
	declaredQueuesMu.RLock()
	_, ok := declaredQueues[queueName]
	declaredQueuesMu.RUnlock()
	if ok {
		return nil
	}

	declaredQueuesMu.Lock()
	defer declaredQueuesMu.Unlock()

	if _, ok := declaredQueues[queueName]; ok {
		return nil
	}

	if _, err := ch.QueueDeclare(
		queueName,
		true,
		false,
		false,
		false,
		nil,
	); err != nil {
		return err
	}

	declaredQueues[queueName] = struct{}{}
	return nil
}

func PublishMessage(userId int64, email, message, queueName, subject string) error {
	notifType := "EMAIL"
	if queueName == utils.PUSH_NOTIF {
		notifType = "PUSH"
	}

	notification := map[string]interface{}{
		"receiver_email":    email,
		"message":           message,
		"receiver_id":       userId,
		"subject":           subject,
		"notification_type": notifType,
	}

	body, err := json.Marshal(notification)
	if err != nil {
		log.Errorf("[PublishMessage-1] Failed to marshal JSON: %v", err)
		return err
	}

	if err := publish(queueName, body); err != nil {
		log.Errorf("[PublishMessage-2] Failed to publish message: %v", err)
		return err
	}

	return nil
}

func publish(queueName string, body []byte) error {
	ch, poolSnapshot, err := borrowChannel()
	if err != nil {
		log.Errorf("[PublishMessage-3] Failed to get RabbitMQ channel: %v", err)
		return err
	}
	defer releaseChannel(ch, poolSnapshot)

	if err := ensureQueueDeclared(ch, queueName); err != nil {
		log.Errorf("[PublishMessage-4] Failed to declare queue: %v", err)
		invalidateRabbitState()
		return err
	}

	if err := ch.Publish(
		"",
		queueName,
		false,
		false,
		amqp.Publishing{
			ContentType: "application/json",
			Body:        body,
		},
	); err == nil {
		return nil
	}

	// One fast retry on fresh connection/channel for transient disconnects.
	invalidateRabbitState()
	chRetry, retryPoolSnapshot, retryErr := borrowChannel()
	if retryErr != nil {
		return fmt.Errorf("retry get channel: %w", retryErr)
	}
	defer releaseChannel(chRetry, retryPoolSnapshot)

	if err := ensureQueueDeclared(chRetry, queueName); err != nil {
		return fmt.Errorf("retry declare queue: %w", err)
	}

	if err := chRetry.Publish(
		"",
		queueName,
		false,
		false,
		amqp.Publishing{
			ContentType: "application/json",
			Body:        body,
		},
	); err != nil {
		return err
	}

	return nil
}
