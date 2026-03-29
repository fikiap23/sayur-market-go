package message

import (
	"encoding/json"
	"fmt"
	"product-service/config"
	"product-service/internal/core/domain/entity"

	"github.com/labstack/gommon/log"
	"github.com/streadway/amqp"
)

type PublishRabbitMQInterface interface {
	PublishProductToQueue(product entity.ProductEntity) error
	DeleteProductFromQueue(productID int64) error
}

type PublishRabbitMQ struct {
	cfg *config.Config
}

func NewPublishRabbitMQ(cfg *config.Config) PublishRabbitMQInterface {
	return &PublishRabbitMQ{cfg: cfg}
}

// DeleteProductFromQueue implements PublishRabbitMQInterface.
func (p *PublishRabbitMQ) DeleteProductFromQueue(productID int64) error {
	conn, err := p.cfg.NewRabbitMQ()
	if err != nil {
		log.Errorf("[DeleteProductFromQueue-1] Failed to connect to RabbitMQ: %v", err)
		return err
	}

	defer conn.Close()

	ch, err := conn.Channel()
	if err != nil {
		log.Errorf("[DeleteProductFromQueue-2] Failed to open a channel: %v", err)
		return err
	}

	defer ch.Close()
	q, err := ch.QueueDeclare(
		p.cfg.PublisherName.ProductDelete,
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		log.Errorf("[DeleteProductFromQueue-3] Failed to declare queue: %v", err)
		return err
	}

	data, _ := json.Marshal(map[string]string{"ProductID": fmt.Sprintf("%d", productID)})
	err = ch.Publish(
		"",
		q.Name,
		false,
		false,
		amqp.Publishing{
			ContentType: "application/json",
			Body:        data,
		},
	)
	if err != nil {
		log.Errorf("[DeleteProductFromQueue-4] Failed to publish message: %v", err)
		return err
	}

	return nil
}

func (p *PublishRabbitMQ) PublishProductToQueue(product entity.ProductEntity) error {
	conn, err := p.cfg.NewRabbitMQ()
	if err != nil {
		log.Errorf("[PublishProductToQueue-1] Failed to connect to RabbitMQ: %v", err)
		return err
	}

	defer conn.Close()

	ch, err := conn.Channel()
	if err != nil {
		log.Errorf("[PublishProductToQueue-2] Failed to open a channel: %v", err)
		return err
	}

	defer ch.Close()

	q, err := ch.QueueDeclare(
		p.cfg.PublisherName.ProductPublish,
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		log.Errorf("[PublishProductToQueue-3] Failed to declare queue: %v", err)
		return err
	}

	data, _ := json.Marshal(product)
	err = ch.Publish(
		"",
		q.Name,
		false,
		false,
		amqp.Publishing{
			ContentType: "application/json",
			Body:        data,
		},
	)
	if err != nil {
		log.Errorf("[PublishProductToQueue-4] Failed to publish message: %v", err)
		return err
	}

	return nil
}
