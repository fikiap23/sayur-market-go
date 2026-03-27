package repository

import (
	"context"
	"errors"
	"fmt"
	"math"
	"notification-service/internal/core/domain/entity"
	"notification-service/internal/core/domain/model"
	"time"

	"github.com/labstack/gommon/log"
	"gorm.io/gorm"
)

type NotificationRepositoryInterface interface {
	GetAll(ctx context.Context, query entity.NotifyQueryString) ([]entity.NotificationEntity, int64, int64, error)
	GetByID(ctx context.Context, notifID uint) (*entity.NotificationEntity, error)
	CreateNotification(ctx context.Context, notification entity.NotificationEntity) error
	MarkAsSent(notifID uint) error
	MarkAsRead(ctx context.Context, notifID uint) error
}

type notificationRepository struct {
	db *gorm.DB
}

// MarkAsRead implements NotificationRepositoryInterface.
func (n *notificationRepository) MarkAsRead(ctx context.Context, notifID uint) error {
	modelNotif := model.Notification{}
	if err := n.db.First(&modelNotif, notifID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			log.Errorf("[MarkAsRead-1] Record not found for notification ID %d", notifID)
			err = errors.New("404")
			return err
		}
		log.Errorf("[MarkAsRead-2] Failed to find notification by ID: %v", err)
		return err
	}

	now := time.Now()
	modelNotif.ReadAt = &now
	if err := n.db.UpdateColumns(&modelNotif).Error; err != nil {
		log.Errorf("[MarkAsRead-3] Failed to save notification: %v", err)
		return err
	}
	return nil
}

// MarkAsSent implements NotificationRepositoryInterface.
func (n *notificationRepository) MarkAsSent(notifID uint) error {
	modelNotif := model.Notification{}

	if err := n.db.First(&modelNotif, notifID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			log.Errorf("[MarkAsSent-1] Record not found for notification ID %d", notifID)
			return err
		}
		log.Errorf("[MarkAsSent-2] Failed to find notification by ID: %v", err)
		return err
	}

	modelNotif.Status = "SENT"

	if err := n.db.UpdateColumns(&modelNotif).Error; err != nil {
		log.Errorf("[MarkAsSent-3] Failed to save notification: %v", err)
		return err
	}

	return nil
}

// CreateNotification implements NotificationRepositoryInterface.
func (n *notificationRepository) CreateNotification(ctx context.Context, notification entity.NotificationEntity) error {
	now := time.Now()
	modelNotif := model.Notification{
		ReceiverID:       notification.ReceiverID,
		Subject:          notification.Subject,
		Status:           notification.Status,
		SentAt:           &now,
		ReadAt:           notification.ReadAt,
		Message:          notification.Message,
		NotificationType: notification.NotificationType,
	}

	if err := n.db.Create(&modelNotif).Error; err != nil {
		log.Errorf("[CreateNotification-1] Failed to create notification: %v", err)
		return err
	}
	return nil
}

// GetByID implements NotificationRepositoryInterface.
func (n *notificationRepository) GetByID(ctx context.Context, notifID uint) (*entity.NotificationEntity, error) {
	modelNotif := model.Notification{}

	if err := n.db.Select("id", "subject", "status", "sent_at", "read_at", "message", "notification_type").First(&modelNotif, notifID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			log.Errorf("[GetByID-1] Record not found for notification ID %d", notifID)
			err = errors.New("404")
			return nil, err
		}
		log.Errorf("[GetByID-2] Failed to find notification by ID: %v", err)
		return nil, err
	}

	return &entity.NotificationEntity{
		ID:               modelNotif.ID,
		Subject:          modelNotif.Subject,
		Status:           modelNotif.Status,
		SentAt:           modelNotif.SentAt,
		ReadAt:           modelNotif.ReadAt,
		Message:          modelNotif.Message,
		NotificationType: modelNotif.NotificationType,
	}, nil
}

// GetAll implements NotificationRepositoryInterface.
func (n *notificationRepository) GetAll(ctx context.Context, queryString entity.NotifyQueryString) ([]entity.NotificationEntity, int64, int64, error) {
	modelNotifes := []model.Notification{}

	var countData int64
	offset := (queryString.Page - 1) * queryString.Limit

	sqlMain := n.db.
		Select("id", "subject", "status", "sent_at").
		Where("subject ILIKE ? OR message ILIKE ? OR status ILIKE ?", "%"+queryString.Search+"%", "%"+queryString.Search+"%", "%"+queryString.Status+"%")

	if queryString.UserID != 0 {
		sqlMain = sqlMain.Where("reciever_id = ?", queryString.UserID)
	}

	if queryString.IsRead {
		sqlMain = sqlMain.Where("read_at IS NOT NULL")
	}

	if err := sqlMain.Model(&modelNotifes).Count(&countData).Error; err != nil {
		log.Errorf("[NotificationRepository-1] GetAll: %v", err)
		return nil, 0, 0, err
	}

	order := fmt.Sprintf("%s %s", queryString.OrderBy, queryString.OrderType)

	totalPage := int(math.Ceil(float64(countData) / float64(queryString.Limit)))
	if err := sqlMain.Order(order).Limit(int(queryString.Limit)).Offset(int(offset)).Find(&modelNotifes).Error; err != nil {
		log.Errorf("[NotificationRepository-2] GetAll: %v", err)
		return nil, 0, 0, err
	}

	if len(modelNotifes) == 0 {
		err := errors.New("404")
		log.Infof("[NotificationRepository-3] GetAll: No notification found")
		return nil, 0, 0, err
	}
	notifEntities := []entity.NotificationEntity{}
	for _, val := range modelNotifes {
		notifEntities = append(notifEntities, entity.NotificationEntity{
			ID:      val.ID,
			Subject: val.Subject,
			Status:  val.Status,
			SentAt:  val.SentAt,
		})
	}

	return notifEntities, countData, int64(totalPage), nil
}

func NewNotificationRepository(db *gorm.DB) NotificationRepositoryInterface {
	return &notificationRepository{db: db}
}
