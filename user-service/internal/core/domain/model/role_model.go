package model

import (
	"time"

	"gorm.io/gorm"
)

type Role struct {
	ID        int64     `gorm:"primaryKey;autoIncrement"`
	Name      string    `gorm:"type:varchar(255);unique;not null"`
	CreatedAt time.Time `gorm:"type:timestamp;default:current_timestamp"`
	UpdatedAt *time.Time
	DeletedAt gorm.DeletedAt `gorm:"index"`
	Users     []User         `gorm:"many2many:user_role"`
}
