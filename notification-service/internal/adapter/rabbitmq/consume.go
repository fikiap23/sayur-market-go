package rabbitmq

import (
	"context"
	"encoding/json"
	"notification-service/config"
	"notification-service/internal/adapter/message"
	"notification-service/internal/adapter/repository"
	"notification-service/internal/core/domain/entity"
	"notification-service/internal/core/service"
	"time"

	"github.com/labstack/gommon/log"
)

type ConsumeRabbitMQInterface interface {
	ConsumeMessage(queueName string) error
}

type consumeRabbitMQ struct {
	emailService        message.MessageEmailInterface
	notifRepository     repository.NotificationRepositoryInterface
	notificationService service.NotificationServiceInterface
}

// ConsumeMessage implements ConsumeRabbitMQInterface.
func (c *consumeRabbitMQ) ConsumeMessage(queueName string) error {
	for {
		conn, err := config.NewConfig().NewRabbitMQ()
		if err != nil {
			log.Errorf("[ConsumeMessage-1] Failed to connect: %v", err)
			time.Sleep(5 * time.Second)
			continue
		}

		ch, err := conn.Channel()
		if err != nil {
			log.Errorf("[ConsumeMessage-2] Failed channel: %v", err)
			conn.Close()
			time.Sleep(5 * time.Second)
			continue
		}

		_, err = ch.QueueDeclare(queueName, true, false, false, false, nil)
		if err != nil {
			log.Errorf("[ConsumeMessage-2] Failed declare queue %s: %v", queueName, err)
			ch.Close()
			conn.Close()
			time.Sleep(5 * time.Second)
			continue
		}

		msgs, err := ch.Consume(queueName, "", true, false, false, false, nil)
		if err != nil {
			log.Errorf("[ConsumeMessage-3] Failed consume: %v", err)
			ch.Close()
			conn.Close()
			time.Sleep(5 * time.Second)
			continue
		}

		// 👇 blocking loop
		for d := range msgs {
			var notificationEntity entity.NotificationEntity

			log.Infof("Received: %s", d.Body)

			if err := json.Unmarshal(d.Body, &notificationEntity); err != nil {
				log.Errorf("Unmarshal error: %v", err)
				continue
			}

			notificationEntity.Status = "PENDING"
			if notificationEntity.NotificationType == "EMAIL" {
				notificationEntity.Status = "SENT"
			}

			if err := c.notifRepository.CreateNotification(context.Background(), notificationEntity); err != nil {
				log.Errorf("DB error: %v", err)
				continue
			}

			go c.SendNotification(notificationEntity)
		}

		// 👇 kalau keluar dari msgs (connection drop)
		log.Warn("Connection lost, reconnecting...")

		ch.Close()
		conn.Close()

		time.Sleep(5 * time.Second)
	}
}

func (c *consumeRabbitMQ) SendNotification(notificationEntity entity.NotificationEntity) {
	switch notificationEntity.NotificationType {
	case "EMAIL":
		err := c.emailService.SendEmailNotif(*notificationEntity.ReceiverEmail, *notificationEntity.Subject, notificationEntity.Message)
		if err != nil {
			log.Errorf("Failed to send email notification: %v", err)
		}
	case "PUSH":
		c.notificationService.SendPushNotification(context.Background(), notificationEntity)
	}
}

func NewConsumeRabbitMQ(emailService message.MessageEmailInterface, notifRepository repository.NotificationRepositoryInterface, notificationService service.NotificationServiceInterface) ConsumeRabbitMQInterface {
	return &consumeRabbitMQ{
		emailService:        emailService,
		notifRepository:     notifRepository,
		notificationService: notificationService,
	}
}
