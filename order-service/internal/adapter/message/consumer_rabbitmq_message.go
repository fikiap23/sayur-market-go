package message

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"order-service/config"
	"order-service/internal/core/domain/entity"

	"github.com/labstack/gommon/log"
)

func ConsumeUpdateStatus() {
	conn, err := config.NewConfig().DialRabbitMQ()
	if err != nil {
		log.Errorf("[ConsumeUpdateStatus-1] Failed to connect to RabbitMQ: %v", err)
	}

	defer conn.Close()

	ch, err := conn.Channel()
	if err != nil {
		log.Errorf("[ConsumeUpdateStatus-2] Failed to open a channel: %v", err)
	}

	defer ch.Close()

	q, err := ch.QueueDeclare(
		config.NewConfig().PublisherName.PublisherUpdateStatus,
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		log.Fatalf("[ConsumeUpdateStatus-3] Failed to declare queue: %v", err)
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
		log.Fatalf("[ConsumeUpdateStatus-4] Failed to register consumer: %v", err)
	}

	log.Info("RabbitMQ Consumer order started...")

	esClient, err := config.NewConfig().InitElasticsearch()
	if err != nil {
		log.Errorf("[ConsumeUpdateStatus-5] Failed initialize Elasticsearch client: %v", err)
	}

	forever := make(chan bool)
	go func() {

		for msqg := range msgs {
			var order map[string]interface{}
			err := json.Unmarshal(msqg.Body, &order)
			if err != nil {
				log.Errorf("[ConsumeUpdateStatus-6] Error decoding message: %v", err)
				continue
			}

			pm, ok := order["status"].(string)
			if !ok || pm == "" {
				log.Errorf("[ConsumeUpdateStatus-7] Invalid or missing status: %v", order["status"])
				continue
			}

			updateScript := map[string]interface{}{
				"script": map[string]interface{}{
					"source": "ctx._source.status = params.status",
					"lang":   "painless",
					"params": map[string]interface{}{
						"status": pm,
					},
				},
			}

			paymentJson, err := json.Marshal(updateScript)
			if err != nil {
				log.Errorf("[ConsumeUpdateStatus-8] Error encoding payment to JSON: %v", err)
				continue
			}

			orderIDStr, ok := order["orderID"].(string)
			if !ok {
				log.Errorf("[ConsumeUpdateStatus-9] Invalid order ID format: %v", order["orderID"])
				continue
			}

			res, err := esClient.Update("orders", orderIDStr, bytes.NewReader(paymentJson))
			if err != nil {
				log.Errorf("[ConsumeUpdateStatus-10] Failed to update payment method in Elasticsearch: %v", err)
			}
			defer res.Body.Close()
			bodyBytes, _ := io.ReadAll(res.Body)
			log.Infof("[ConsumeUpdateStatus-11] Elasticsearch response: %s", string(bodyBytes))
		}
	}()

	log.Infof("[ConsumeUpdateStatus-11] Waiting for messages. To exit press CTRL+C")
	<-forever
}

func ConsumePaymentSuccess() {
	conn, err := config.NewConfig().DialRabbitMQ()
	if err != nil {
		log.Errorf("[consumePaymentSuccess-1] Failed to connect to RabbitMQ: %v", err)
	}

	defer conn.Close()

	ch, err := conn.Channel()
	if err != nil {
		log.Errorf("[consumePaymentSuccess-2] Failed to open a channel: %v", err)
	}

	defer ch.Close()

	q, err := ch.QueueDeclare(
		config.NewConfig().PublisherName.PublisherPaymentSuccess,
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		log.Fatalf("[consumePaymentSuccess-3] Failed to declare queue: %v", err)
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
		log.Fatalf("[consumePaymentSuccess-4] Failed to register consumer: %v", err)
	}

	log.Info("RabbitMQ Consumer order started...")

	esClient, err := config.NewConfig().InitElasticsearch()
	if err != nil {
		log.Errorf("[StartOrderConsumer-5] Failed initialize Elasticsearch client: %v", err)
	}

	forever := make(chan bool)
	go func() {

		for msqg := range msgs {
			var payment map[string]interface{}
			err := json.Unmarshal(msqg.Body, &payment)
			if err != nil {
				log.Errorf("[consumePaymentSuccess-5] Error decoding message: %v", err)
				continue
			}

			pm, ok := payment["paymentMethod"].(string)
			if !ok || pm == "" {
				log.Errorf("[consumePaymentSuccess-6] Invalid or missing paymentMethod: %v", payment["paymentMethod"])
				continue
			}

			updateScript := map[string]interface{}{
				"script": map[string]interface{}{
					"source": "ctx._source.PaymentMethod = params.payment_method",
					"lang":   "painless",
					"params": map[string]interface{}{
						"payment_method": pm,
					},
				},
			}

			paymentJson, err := json.Marshal(updateScript)
			if err != nil {
				log.Errorf("[consumePaymentSuccess-7] Error encoding payment to JSON: %v", err)
				continue
			}

			orderIDStr, ok := payment["orderID"].(string)
			if !ok {
				log.Errorf("[consumePaymentSuccess-7] Invalid order ID format: %v", payment["orderID"])
				continue
			}

			res, err := esClient.Update("orders", orderIDStr, bytes.NewReader(paymentJson))
			if err != nil {
				log.Errorf("[consumePaymentSuccess-8] Failed to update payment method in Elasticsearch: %v", err)
			}
			defer res.Body.Close()
			bodyBytes, _ := io.ReadAll(res.Body)
			log.Infof("[consumePaymentSuccess-9] Elasticsearch response: %s", string(bodyBytes))
		}
	}()

	log.Infof("[consumePaymentSuccess-8] Waiting for messages. To exit press CTRL+C")
	<-forever
}

func ConsumeDeleteOrder() {
	conn, err := config.NewConfig().DialRabbitMQ()
	if err != nil {
		log.Errorf("[ConsumeDeleteOrder-1] Failed to connect to RabbitMQ: %v", err)
	}

	defer conn.Close()

	ch, err := conn.Channel()
	if err != nil {
		log.Errorf("[ConsumeDeleteOrder-2] Failed to open a channel: %v", err)
	}

	defer ch.Close()

	q, err := ch.QueueDeclare(
		config.NewConfig().PublisherName.PublisherDeleteOrder,
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		log.Fatalf("[ConsumeDeleteOrder-3] Failed to declare queue: %v", err)
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
		log.Fatalf("[ConsumeDeleteOrder-4] Failed to register consumer: %v", err)
	}

	log.Info("RabbitMQ Consumer order started...")

	esClient, err := config.NewConfig().InitElasticsearch()
	if err != nil {
		log.Errorf("[ConsumeDeleteOrder-5] Failed initialize Elasticsearch client: %v", err)
	}

	forever := make(chan bool)
	go func() {

		for msqg := range msgs {
			var order map[string]string
			err := json.Unmarshal(msqg.Body, &order)
			if err != nil {
				log.Errorf("[ConsumeDeleteOrder-5] Error decoding message: %v", err)
				continue
			}

			orderID := order["orderID"]

			res, err := esClient.Delete("orders", orderID)
			if err != nil {
				log.Errorf("[ConsumeDeleteOrder-7] Failed to update payment method in Elasticsearch: %v", err)
			}
			defer res.Body.Close()
		}
	}()

	log.Infof("[ConsumeDeleteOrder-8] Waiting for messages. To exit press CTRL+C")
	<-forever
}

func StartOrderConsumer() {
	conn, err := config.NewConfig().DialRabbitMQ()
	if err != nil {
		log.Errorf("[StartOrderConsumer-1] Failed to connect to RabbitMQ: %v", err)
		return
	}

	defer conn.Close()

	ch, err := conn.Channel()
	if err != nil {
		log.Errorf("[StartOrderConsumer-2] Failed to open a channel: %v", err)
		return
	}

	defer ch.Close()

	q, err := ch.QueueDeclare(
		config.NewConfig().PublisherName.OrderPublish,
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		log.Fatalf("[StartOrderConsumer-3] Failed to declare queue: %v", err)
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
		log.Fatalf("[StartOrderConsumer-4] Failed to register consumer: %v", err)
		return
	}

	log.Info("RabbitMQ Consumer order started...")

	esClient, err := config.NewConfig().InitElasticsearch()
	if err != nil {
		log.Errorf("[StartOrderConsumer-5] Failed initialize Elasticsearch client: %v", err)
		return
	}

	forever := make(chan bool)
	go func() {
		for d := range msgs {
			var order entity.OrderEntity
			err := json.Unmarshal(d.Body, &order)
			if err != nil {
				log.Errorf("[StartOrderConsumer-6] Error decoding message: %v", err)
				continue
			}

			// Convert order struct ke JSON
			orderJSON, err := json.Marshal(order)
			if err != nil {
				log.Errorf("[StartOrderConsumer-7] Error encoding order to JSON: %v", err)
				continue
			}

			// Indexing ke Elasticsearch
			res, err := esClient.Index(
				"orders",                   // Nama index di Elasticsearch
				bytes.NewReader(orderJSON), // Data JSON
				esClient.Index.WithDocumentID(fmt.Sprintf("%d", order.ID)), // ID dokumen
				esClient.Index.WithContext(context.Background()),
				esClient.Index.WithRefresh("true"),
			)
			if err != nil {
				log.Errorf("[StartOrderConsumer-8] Error indexing to Elasticsearch: %v", err)
				continue
			}
			defer res.Body.Close()

			body, _ := io.ReadAll(res.Body)

			log.Infof("[StartOrderConsumer-9] Order %d berhasil diindex ke Elasticsearch %v", order.ID, string(body))
		}
	}()

	log.Infof("[StartOrderConsumer-10] Waiting for messages. To exit press CTRL+C")
	<-forever
}
