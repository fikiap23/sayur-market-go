package model

import "time"

type User struct {
	ID         int64     `gorm:"primaryKey;autoIncrement"`
	Name       string    `gorm:"type:varchar(255);not null"`
	Email      string    `gorm:"type:varchar(255);unique;not null;index:idx_users_email"`
	Password   string    `gorm:"type:varchar(255);not null"`
	Phone      string    `gorm:"type:varchar(17)"`
	Photo      string    `gorm:"type:varchar(255)"`
	Address    string    `gorm:"type:text"`
	Lat        string    `gorm:"type:varchar(50)"`
	Lng        string    `gorm:"type:varchar(50)"`
	IsVerified bool      `gorm:"type:boolean;default:false;index:idx_users_is_verified"`
	CreatedAt  time.Time `gorm:"type:timestamp;default:current_timestamp"`
	UpdatedAt  *time.Time
	DeletedAt  *time.Time `gorm:"index"`
	Roles      []Role     `gorm:"many2many:user_role;"`
}
