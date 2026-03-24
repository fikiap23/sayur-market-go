package model

import (
	"time"

	"gorm.io/gorm"
)

type VerificationToken struct {
	ID        int64     `gorm:"primaryKey;autoIncrement"`
	UserID    int64     `gorm:"not null;index:idx_verification_users_user_id"`
	Token     string    `gorm:"type:varchar(255);not null"`
	TokenType string    `gorm:"type:varchar(20);not null"`
	ExpiresAt time.Time `gorm:"type:timestamp;not null"`
	CreatedAt time.Time `gorm:"type:timestamp;default:current_timestamp"`
	UpdatedAt *time.Time
	DeletedAt gorm.DeletedAt `gorm:"index"`

	// Relasi ke User
	User User `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE"`
}
