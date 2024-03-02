package gamesrv

import (
	"log/slog"
	"sync"
	"time"

	"github.com/gorcon/rcon"
	"github.com/kofuk/premises/common/retry"
)

type Rcon struct {
	addr     string
	password string
	mu       sync.Mutex
}

func NewRcon(addr, password string) *Rcon {
	return &Rcon{
		addr:     addr,
		password: password,
	}
}

func (self *Rcon) connect() (*rcon.Conn, error) {
	conn, err := rcon.Dial(self.addr, self.password)
	if err != nil {
		return nil, err
	}
	return conn, nil
}

func (self *Rcon) waitConnect() (*rcon.Conn, error) {
	var conn *rcon.Conn
	err := retry.Retry(func() error {
		var err error
		conn, err = self.connect()
		if err != nil {
			return err
		}
		return nil
	}, 20*time.Minute)
	if err != nil {
		return nil, err
	}

	return conn, nil
}

func (self *Rcon) Execute(cmd string) (string, error) {
	self.mu.Lock()
	defer self.mu.Unlock()

	conn, err := self.waitConnect()
	if err != nil {
		return "", err
	}
	defer conn.Close()

	slog.Info("Executing rcon", slog.String("command", cmd))
	resp, err := conn.Execute(cmd)
	if err != nil {
		return "", err
	}
	slog.Info("Rcon response received", slog.String("command", cmd), slog.String("response", resp))

	return resp, nil
}
