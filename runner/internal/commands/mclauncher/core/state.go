package core

//go:generate go run go.uber.org/mock/mockgen@v0.5.0 -destination state_mock.go -package core . StateRepository

type StateRepository interface {
	SetState(key string, state any) error
	GetState(key string) any
}
