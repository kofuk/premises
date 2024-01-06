package model

import (
	"gorm.io/gorm"
)

type User struct {
	gorm.Model
	Name          string `gorm:"type:varchar(32);not null;uniqueIndex"`
	Password      string `gorm:"type:varchar(64);not null"`
	AddedByUserID *uint
	AddedBy       *User `gorm:"foreignKey:AddedByUserID"`
	Initialized   bool  `gorm:"not null"`
}
