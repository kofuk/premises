package rcon

//go:generate go tool mockgen -destination executor_mock.go -package rcon . RconExecutorInterface

import (
	"log/slog"
	"sync"
	"time"

	"github.com/gorcon/rcon"
	"github.com/kofuk/premises/internal/retry"
)

type RconExecutorInterface interface {
	Exec(cmd string) (string, error)
}

type RconExecutor struct {
	addr     string
	password string
	mu       sync.Mutex
}

var _ RconExecutorInterface = (*RconExecutor)(nil)

func NewRconExecutor(addr, password string) *RconExecutor {
	return &RconExecutor{
		addr:     addr,
		password: password,
	}
}

func (r *RconExecutor) connect() (*rcon.Conn, error) {
	conn, err := rcon.Dial(r.addr, r.password)
	if err != nil {
		return nil, err
	}
	return conn, nil
}

func (r *RconExecutor) waitConnect() (*rcon.Conn, error) {
	conn, err := retry.Retry(func() (*rcon.Conn, error) {
		return r.connect()
	}, 20*time.Minute)
	if err != nil {
		return nil, err
	}

	return conn, nil
}

func (r *RconExecutor) Exec(cmd string) (string, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	conn, err := r.waitConnect()
	if err != nil {
		return "", err
	}
	defer conn.Close()

	slog.Debug("Executing rcon", slog.String("command", cmd))
	resp, err := conn.Execute(cmd)
	if err != nil {
		return "", err
	}
	slog.Debug("Rcon response received", slog.String("command", cmd), slog.String("response", resp))

	return resp, nil
}
