package core

import "context"

//go:generate go run go.uber.org/mock/mockgen@v0.5.0 -destination state_mock.go -package core . StateRepository

type StateRepository interface {
	SetState(ctx context.Context, key string, state string) error
	RemoveState(ctx context.Context, key string) error
	GetState(ctx context.Context, key string) (string, error)
}
