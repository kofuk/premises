package model

import (
	"gorm.io/gorm"
)

type User struct {
	gorm.Model
	Name          string `gorm:"type:varchar(32);not null;uniqueIndex"`
	Password      string `gorm:"type:varchar(64);not null"`
	AddedByUserID *uint
	AddedBy       *User        `gorm:"foreignKey:AddedByUserID"`
	Credentials   []Credential `gorm:"foreignKey:OwnerID"`
	Initialized   bool         `gorm:"not null"`
}

type Credential struct {
	gorm.Model
	OwnerID                uint   `gorm:"not null"`
	UUID                   string `gorm:"type:varchar(36);not null;unique"`
	KeyName                string `gorm:"type:varchar(128);not null"`
	CredentialID           []byte `gorm:"type:bytea;not null"`
	PublicKey              []byte `gorm:"type:bytea;not null"`
	AttestationType        string `gorm:"type:varchar(16);not null"`
	AuthenticatorAAGUID    []byte `gorm:"type:bytea;not null"`
	AuthenticatorSignCount uint32 `gorm:"not null"`
}

