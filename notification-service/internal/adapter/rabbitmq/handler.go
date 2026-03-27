package rabbitmq

import (
	"context"
	"encoding/json"
	"fmt"

	"notification-service/internal/adapter/message"
	"notification-service/internal/adapter/repository"
	"notification-service/internal/core/domain/entity"
	"notification-service/internal/core/service"

	"github.com/rs/zerolog"
)

// NotificationHandler processes notification messages from RabbitMQ.
// It persists the notification to the database and dispatches delivery
// (email / push) as a best-effort side effect.
type NotificationHandler struct {
	emailService        message.MessageEmailInterface
	notifRepository     repository.NotificationRepositoryInterface
	notificationService service.NotificationServiceInterface
	logger              zerolog.Logger
}

func NewNotificationHandler(
	emailService message.MessageEmailInterface,
	notifRepo repository.NotificationRepositoryInterface,
	notifService service.NotificationServiceInterface,
	logger zerolog.Logger,
) *NotificationHandler {
	return &NotificationHandler{
		emailService:        emailService,
		notifRepository:     notifRepo,
		notificationService: notifService,
		logger:              logger.With().Str("component", "notification_handler").Logger(),
	}
}

// Handle deserialises the raw message, persists a notification record, and
// attempts delivery. Returning an error triggers the consumer's retry logic;
// delivery failures after a successful DB persist are logged but do not cause
// a retry (the record already exists).
func (h *NotificationHandler) Handle(ctx context.Context, body []byte) error {
	var notif entity.NotificationEntity
	if err := json.Unmarshal(body, &notif); err != nil {
		return fmt.Errorf("unmarshal notification: %w", err)
	}

	notif.Status = "PENDING"
	if notif.NotificationType == "EMAIL" {
		notif.Status = "SENT"
	}

	if err := h.notifRepository.CreateNotification(ctx, notif); err != nil {
		return fmt.Errorf("create notification: %w", err)
	}

	if err := h.sendNotification(ctx, notif); err != nil {
		h.logger.Warn().
			Err(err).
			Str("type", notif.NotificationType).
			Msg("notification delivery failed after DB persist")
	}

	return nil
}

func (h *NotificationHandler) sendNotification(ctx context.Context, notif entity.NotificationEntity) error {
	switch notif.NotificationType {
	case "EMAIL":
		if notif.ReceiverEmail == nil || notif.Subject == nil {
			return fmt.Errorf("email notification missing receiver_email or subject")
		}
		return h.emailService.SendEmailNotif(*notif.ReceiverEmail, *notif.Subject, notif.Message)
	case "PUSH":
		h.notificationService.SendPushNotification(ctx, notif)
		return nil
	default:
		h.logger.Warn().Str("type", notif.NotificationType).Msg("unknown notification type")
		return nil
	}
}
