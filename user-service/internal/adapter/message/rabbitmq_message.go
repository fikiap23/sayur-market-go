package message

import (
	"context"
	"encoding/json"
	"fmt"

	"user-service/internal/adapter/rabbitmq"
	"user-service/utils"

	"github.com/rs/zerolog"
)

// EventPublisher defines the contract for publishing notification events
// to RabbitMQ. Inject this interface into the service layer.
type EventPublisher interface {
	PublishNotification(ctx context.Context, userId int64, email, message, queueName, subject string) error
}

type rabbitMQEventPublisher struct {
	publisher *rabbitmq.Publisher
	logger    zerolog.Logger
}

func NewEventPublisher(publisher *rabbitmq.Publisher, logger zerolog.Logger) EventPublisher {
	return &rabbitMQEventPublisher{
		publisher: publisher,
		logger:    logger.With().Str("component", "event_publisher").Logger(),
	}
}

func (p *rabbitMQEventPublisher) PublishNotification(ctx context.Context, userId int64, email, msg, queueName, subject string) error {
	notifType := "EMAIL"
	if queueName == utils.PUSH_NOTIF {
		notifType = "PUSH"
	}

	payload := map[string]interface{}{
		"receiver_email":    email,
		"message":           msg,
		"receiver_id":       userId,
		"subject":           subject,
		"notification_type": notifType,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal notification payload: %w", err)
	}

	if err := p.publisher.Publish(ctx, queueName, body); err != nil {
		p.logger.Error().Err(err).
			Str("queue", queueName).
			Int64("user_id", userId).
			Msg("failed to publish notification")
		return err
	}

	p.logger.Debug().
		Str("queue", queueName).
		Int64("user_id", userId).
		Msg("notification published")
	return nil
}
