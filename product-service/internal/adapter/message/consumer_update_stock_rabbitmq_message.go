package message

import (
	"encoding/json"
	"product-service/config"
	"product-service/internal/core/domain/entity"
	"product-service/internal/core/domain/model"

	"github.com/labstack/gommon/log"
)

// StartUpdateStockConsumer implements consumerUpdateStockInterface.
func StartUpdateStockConsumer() {
	db, err := config.NewConfig().ConnectionPostgres()
	if err != nil {
		log.Errorf("[StartUpdateStockConsumer-1] Failed to connect to PostgreSQL: %v", err)
		return
	}

	conn, err := config.NewConfig().NewRabbitMQ()
	if err != nil {
		log.Errorf("[StartUpdateStockConsumer-1] Failed to connect to RabbitMQ: %v", err)
		return
	}

	defer conn.Close()

	ch, err := conn.Channel()
	if err != nil {
		log.Errorf("[StartUpdateStockConsumer-2] Failed to open a channel: %v", err)
		return
	}

	defer ch.Close()

	q, err := ch.QueueDeclare(
		config.NewConfig().PublisherName.ProductUpdateStock,
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		log.Fatalf("[StartConsumer-3] Failed to declare queue: %v", err)
		return
	}

	msgs, err := ch.Consume(
		q.Name,
		"",
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		log.Fatalf("[StartConsumer-4] Failed to register consumer: %v", err)
		return
	}

	log.Info("RabbitMQ Consumer started...")

	for msg := range msgs {
		var orderItem entity.PublishOrderItemEntity
		if err := json.Unmarshal(msg.Body, &orderItem); err != nil {
			log.Errorf("[StartUpdateStockConsumer-5] Failed to decode message: %v", err)
			continue
		}

		// Simulasi update stok
		var product model.Product
		if err := db.DB.First(&product, orderItem.ProductID).Error; err != nil {
			log.Errorf("[StartUpdateStockConsumer-6] Failed to find product: %v", err)
			continue
		}

		if product.Stock < int(orderItem.Quantity) {
			log.Errorf("[StartUpdateStockConsumer-7] Stock not enough")
			continue
		}

		product.Stock -= int(orderItem.Quantity)
		if err := db.DB.Save(&product).Error; err != nil {
			log.Errorf("[StartUpdateStockConsumer-8] Failed to update stock: %v", err)
			continue
		}
		log.Printf("Mengurangi stok produk %d sebanyak %d", orderItem.ProductID, orderItem.Quantity)
	}
}
