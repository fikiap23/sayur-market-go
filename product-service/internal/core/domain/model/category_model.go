package model

import (
	"time"

	"gorm.io/gorm"
)

type Category struct {
	ID          int64          `gorm:"primaryKey;autoIncrement"`
	ParentID    *int64         `gorm:"column:parent_id"`
	Name        string         `gorm:"column:name;not null"`
	Icon        string         `gorm:"column:icon;not null"`
	Status      bool           `gorm:"column:status;default:true"`
	Slug        string         `gorm:"column:slug;unique;not null"`
	Description *string        `gorm:"column:description"`
	CreatedAt   time.Time      `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt   *time.Time     `gorm:"column:updated_at;autoUpdateTime"`
	DeletedAt   gorm.DeletedAt `gorm:"column:deleted_at;index"`

	// relations
	Products []Product `gorm:"foreignKey:CategorySlug;references:Slug"`

	Parent   *Category  `gorm:"foreignKey:ParentID"`
	Children []Category `gorm:"foreignKey:ParentID"`
}
