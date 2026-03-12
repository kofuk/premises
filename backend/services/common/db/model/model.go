package model

import (
	"time"

	"github.com/uptrace/bun"
)

type User struct {
	bun.BaseModel `bun:"table:users"`

	ID            uint         `bun:"id,pk,autoincrement"`
	CreatedAt     time.Time    `bun:"created_at,nullzero,notnull,default:current_timestamp"`
	UpdatedAt     time.Time    `bun:"updated_at,nullzero,notnull,default:current_timestamp"`
	DeletedAt     bun.NullTime `bun:"deleted_at"`
	Name          string       `bun:"name,type:varchar(32),notnull,unique"`
	Password      string       `bun:"password,type:varchar(64),notnull"`
	AddedByUserID *uint        `bun:"added_by_user_id"`
	Initialized   bool         `bun:"initialized,notnull"`
}
