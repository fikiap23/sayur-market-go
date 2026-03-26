package message

import (
	"encoding/json"
	"user-service/config"
	"user-service/utils"

	"github.com/labstack/gommon/log"
	"github.com/streadway/amqp"
)

func PublishMessage(userId int64, email, message, queueName, subject string) error {
	conn, err := config.NewConfig().NewRabbitMQ()
	if err != nil {
		log.Errorf("[PublishMessage-1] Failed to connect to RabbitMQ: %v", err)
		return err
	}

	defer conn.Close()

	ch, err := conn.Channel()
	if err != nil {
		log.Errorf("[PublishMessage-2] Failed to open a channel: %v", err)
		return err
	}

	defer ch.Close()

	queue, err := ch.QueueDeclare(
		queueName,
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		log.Errorf("[PublishMessage-3] Failed to declare a queue: %v", err)
		return err
	}

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
		log.Errorf("[PublishMessage-4] Failed to marshal JSON: %v", err)
		return err
	}

	return ch.Publish(
		"",
		queue.Name,
		false,
		false,
		amqp.Publishing{
			ContentType: "application/json",
			Body:        body,
		},
	)
}
