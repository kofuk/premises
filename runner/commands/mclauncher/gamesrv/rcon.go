package gamesrv

import (
	"errors"
	"log/slog"
	"math/rand"
	"sync"
	"time"

	"github.com/gorcon/rcon"
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
	totalWait := 0
	var err error
	for i := 0; i < 500; i++ {
		var conn *rcon.Conn
		conn, err = self.connect()
		if err == nil {
			return conn, nil
		}

		if totalWait >= 600 {
			return nil, errors.New("Timed out")
		}

		slog.Info("Failed to connect rcon; retrying...", slog.Any("error", err))

		waitDur := (1 << i) + rand.Intn(5)
		time.Sleep(time.Duration(waitDur) * time.Second)
		totalWait += waitDur
	}
	return nil, err
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
