package core

import "context"

//go:generate go tool mockgen -destination state_mock.go -package core . StateRepository

type StateRepository interface {
	SetState(ctx context.Context, key string, state string) error
	RemoveState(ctx context.Context, key string) error
	GetState(ctx context.Context, key string) (string, error)
}
