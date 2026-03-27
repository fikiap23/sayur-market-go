package model

import (
	"time"

	"gorm.io/gorm"
)

type UserRole struct {
	ID        int64     `gorm:"primaryKey;autoIncrement"`
	RoleID    int64     `gorm:"not null"`
	UserID    int64     `gorm:"not null"`
	CreatedAt time.Time `gorm:"type:timestamp;default:current_timestamp"`
	UpdatedAt *time.Time
	DeletedAt gorm.DeletedAt `gorm:"index"`
}

type Tabler interface {
	TableName() string
}

func (UserRole) TableName() string {
	return "user_role"
}
