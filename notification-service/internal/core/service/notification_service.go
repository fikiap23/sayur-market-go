package service

import (
	"context"
	"notification-service/internal/adapter/repository"
	"notification-service/internal/core/domain/entity"
	"notification-service/utils"

	"github.com/labstack/gommon/log"
)

type NotificationServiceInterface interface {
	GetAll(ctx context.Context, queryString entity.NotifyQueryString) ([]entity.NotificationEntity, int64, int64, error)
	GetByID(ctx context.Context, notifID uint) (*entity.NotificationEntity, error)
	SendPushNotification(ctx context.Context, notification entity.NotificationEntity)
	MarkAsRead(ctx context.Context, notifID uint) error
}

type NotificationService struct {
	repo repository.NotificationRepositoryInterface
}

// MarkAsRead implements NotificationServiceInterface.
func (n *NotificationService) MarkAsRead(ctx context.Context, notifID uint) error {
	return n.repo.MarkAsRead(ctx, notifID)
}

// SendPushNotification implements NotificationServiceInterface.
func (n *NotificationService) SendPushNotification(ctx context.Context, notif entity.NotificationEntity) {
	if notif.ReceiverID == nil {
		return
	}
	conn := utils.GetWebSocketConn(*notif.ReceiverID)
	if conn == nil {
		log.Errorf("[SendPushNotification-1] WebSocket connection not found for user %d", *notif.ReceiverID)
		return
	}

	msg := map[string]interface{}{
		"type":    notif.NotificationType,
		"subject": notif.Subject,
		"message": notif.Message,
		"sent_at": notif.SentAt,
	}

	if err := conn.WriteJSON(msg); err != nil {
		log.Errorf("[SendPushNotification-2] Failed to send WebSocket notification: %v", err)
	}

	if err := n.repo.MarkAsSent(notif.ID); err != nil {
		log.Errorf("[SendPushNotification-3] Failed to mark notification as sent: %v", err)
	}
}

// GetByID implements NotificationServiceInterface.
func (n *NotificationService) GetByID(ctx context.Context, notifID uint) (*entity.NotificationEntity, error) {
	return n.repo.GetByID(ctx, notifID)
}

// GetAll implements NotificationServiceInterface.
func (n *NotificationService) GetAll(ctx context.Context, queryString entity.NotifyQueryString) ([]entity.NotificationEntity, int64, int64, error) {
	return n.repo.GetAll(ctx, queryString)
}

func NewNotificationService(repo repository.NotificationRepositoryInterface) NotificationServiceInterface {
	return &NotificationService{repo: repo}
}
