package entity

import (
	"time"
)

type NotificationEntity struct {
	NotificationType string     `json:"notification_type"`
	ReceiverID       *int       `json:"receiver_id"`
	Subject          *string    `json:"subject"`
	Message          string     `json:"message"`
	ReceiverEmail    *string    `json:"receiver_email"`
	SentAt           *time.Time `json:"sent_at"`
	ReadAt           *time.Time `json:"read_at"`
	Status           string     `json:"status"`
	ID               uint       `json:"id"`
}

type NotifyQueryString struct {
	Page      int64
	Limit     int64
	Search    string
	Status    string
	OrderBy   string
	OrderType string
	UserID    uint
	IsRead    bool
}
