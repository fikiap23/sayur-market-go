package message

import (
	"context"
	"encoding/json"
	"fmt"

	"product-service/internal/adapter/rabbitmq"
	"product-service/internal/core/domain/entity"

	"github.com/rs/zerolog"
)

// PublishRabbitMQInterface is the contract used by the service layer.
type PublishRabbitMQInterface interface {
	PublishProductToQueue(product entity.ProductEntity) error
	DeleteProductFromQueue(productID int64) error
}

type publishRabbitMQ struct {
	publisher         *rabbitmq.Publisher
	publishQueueName  string
	deleteQueueName   string
	logger            zerolog.Logger
}

func NewPublishRabbitMQ(publisher *rabbitmq.Publisher, publishQueue, deleteQueue string, logger zerolog.Logger) PublishRabbitMQInterface {
	return &publishRabbitMQ{
		publisher:        publisher,
		publishQueueName: publishQueue,
		deleteQueueName:  deleteQueue,
		logger:           logger.With().Str("component", "product_publisher").Logger(),
	}
}

func (p *publishRabbitMQ) PublishProductToQueue(product entity.ProductEntity) error {
	body, err := json.Marshal(product)
	if err != nil {
		return fmt.Errorf("marshal product: %w", err)
	}

	if err := p.publisher.Publish(context.Background(), p.publishQueueName, body); err != nil {
		p.logger.Error().Err(err).Int64("product_id", product.ID).Msg("failed to publish product")
		return err
	}

	p.logger.Debug().Int64("product_id", product.ID).Msg("product published")
	return nil
}

func (p *publishRabbitMQ) DeleteProductFromQueue(productID int64) error {
	body, _ := json.Marshal(map[string]string{
		"ProductID": fmt.Sprintf("%d", productID),
	})

	if err := p.publisher.Publish(context.Background(), p.deleteQueueName, body); err != nil {
		p.logger.Error().Err(err).Int64("product_id", productID).Msg("failed to publish product delete")
		return err
	}

	p.logger.Debug().Int64("product_id", productID).Msg("product delete published")
	return nil
}
