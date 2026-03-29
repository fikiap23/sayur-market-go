package message

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"product-service/config"
	"product-service/internal/core/domain/entity"

	"github.com/labstack/gommon/log"
)

func StartDeleteOrderConsumer() {
	conn, err := config.NewConfig().NewRabbitMQ()
	if err != nil {
		log.Errorf("[StartDeleteOrderConsumer-1] Failed to connect to RabbitMQ: %v", err)
	}

	defer conn.Close()

	ch, err := conn.Channel()
	if err != nil {
		log.Errorf("[StartDeleteOrderConsumer-2] Failed to open a channel: %v", err)
	}

	defer ch.Close()

	q, err := ch.QueueDeclare(
		config.NewConfig().PublisherName.ProductDelete,
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		log.Fatalf("[StartDeleteOrderConsumer-3] Failed to declare queue: %v", err)
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
		log.Fatalf("[StartDeleteOrderConsumer-4] Failed to register consumer: %v", err)
	}

	log.Info("RabbitMQ Consumer started...")

	esClient, err := config.NewConfig().InitElasticsearch()
	if err != nil {
		log.Errorf("[StartDeleteOrderConsumer-5] Failed initialize Elasticsearch client: %v", err)
	}

	forever := make(chan bool)
	go func() {
		for d := range msgs {
			var data map[string]string
			err := json.Unmarshal(d.Body, &data)
			if err != nil {
				log.Errorf("[StartDeleteOrderConsumer-6] Error decoding message: %v", err)
				continue
			}

			productID := data["ProductID"]

			res, err := esClient.Delete("products", productID)
			if err != nil {
				log.Errorf("[StartDeleteOrderConsumer-8] Error indexing to Elasticsearch: %v", err)
				continue
			}
			defer res.Body.Close()
		}
	}()

	log.Infof("[StartDeleteOrderConsumer-10] Waiting for messages. To exit press CTRL+C")
	<-forever
}

func StartConsumer() {
	conn, err := config.NewConfig().NewRabbitMQ()
	if err != nil {
		log.Errorf("[StartConsumer-1] Failed to connect to RabbitMQ: %v", err)
	}

	defer conn.Close()

	ch, err := conn.Channel()
	if err != nil {
		log.Errorf("[StartConsumer-2] Failed to open a channel: %v", err)
	}

	defer ch.Close()

	q, err := ch.QueueDeclare(
		config.NewConfig().PublisherName.ProductPublish,
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		log.Fatalf("[StartConsumer-3] Failed to declare queue: %v", err)
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
	}

	log.Info("RabbitMQ Consumer started...")

	esClient, err := config.NewConfig().InitElasticsearch()
	if err != nil {
		log.Errorf("[StartConsumer-5] Failed initialize Elasticsearch client: %v", err)
	}

	forever := make(chan bool)
	go func() {
		for d := range msgs {
			var product entity.ProductEntity
			err := json.Unmarshal(d.Body, &product)
			if err != nil {
				log.Errorf("[StartConsumer-6] Error decoding message: %v", err)
				continue
			}

			// Convert product struct ke JSON
			productJSON, err := json.Marshal(product)
			if err != nil {
				log.Errorf("[StartConsumer-7] Error encoding product to JSON: %v", err)
				continue
			}

			// Indexing ke Elasticsearch
			res, err := esClient.Index(
				"products",                   // Nama index di Elasticsearch
				bytes.NewReader(productJSON), // Data JSON
				esClient.Index.WithDocumentID(fmt.Sprintf("%d", product.ID)), // ID dokumen
				esClient.Index.WithContext(context.Background()),
				esClient.Index.WithRefresh("true"),
			)
			if err != nil {
				log.Errorf("[StartConsumer-8] Error indexing to Elasticsearch: %v", err)
				continue
			}
			defer res.Body.Close()

			body, _ := io.ReadAll(res.Body)

			log.Infof("[StartConsumer-9] Product %d berhasil diindex ke Elasticsearch %v", product.ID, string(body))
		}
	}()

	log.Infof("[StartConsumer-10] Waiting for messages. To exit press CTRL+C")
	<-forever
}
