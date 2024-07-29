package game

import (
	"errors"
	"fmt"
	"log/slog"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gorcon/rcon"
	"github.com/kofuk/premises/internal/retry"
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

func (r *Rcon) connect() (*rcon.Conn, error) {
	conn, err := rcon.Dial(r.addr, r.password)
	if err != nil {
		return nil, err
	}
	return conn, nil
}

func (r *Rcon) waitConnect() (*rcon.Conn, error) {
	var conn *rcon.Conn
	err := retry.Retry(func() error {
		var err error
		conn, err = r.connect()
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

func (r *Rcon) Execute(cmd string) (string, error) {
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

func (r *Rcon) SaveAll() error {
	if _, err := r.Execute("save-all"); err != nil {
		return err
	}
	return nil
}

func (r *Rcon) AddToWhiteList(player string) error {
	if _, err := r.Execute(fmt.Sprintf("whitelist add %s", player)); err != nil {
		return fmt.Errorf("%s: Failed to add to whitelist: %w", player, err)
	}
	return nil
}

func (r *Rcon) AddToOp(player string) error {
	if _, err := r.Execute(fmt.Sprintf("op %s", player)); err != nil {
		return fmt.Errorf("%s: Failed to add to op: %w", player, err)
	}
	return nil
}

func (r *Rcon) Say(message string) error {
	if _, err := r.Execute(fmt.Sprintf("tellraw @a \"%s\"", message)); err != nil {
		return err
	}

	return nil
}

func parseSeed(seedOutput string) (string, error) {
	if len(seedOutput) < 8 || seedOutput[:7] != "Seed: [" || seedOutput[len(seedOutput)-1] != ']' {
		return "", errors.New("failed to retrieve seed")
	}

	return seedOutput[7 : len(seedOutput)-1], nil
}

func (r *Rcon) Seed() (string, error) {
	seed, err := r.Execute("seed")
	if err != nil {
		return "", err
	}

	return parseSeed(seed)
}

func (r *Rcon) Stop() error {
	if _, err := r.Execute("stop"); err != nil {
		return err
	}
	return nil
}

type PlayerList []string

var (
	playerListRegexp = regexp.MustCompile("^There are ([0-9]+) of a max of [0-9]+ players online: (.*)$")
)

func parsePlayerList(listOutput string) (PlayerList, error) {
	match := playerListRegexp.FindStringSubmatch(listOutput)
	if match == nil {
		return nil, errors.New("invalid /list output")
	}

	count, _ := strconv.Atoi(match[1]) // Error should not be occurred

	var players []string
	for _, p := range strings.Split(match[2], ", ") {
		if p != "" {
			players = append(players, strings.Trim(p, " "))
		}
	}

	if count != len(players) {
		return nil, errors.New("player count mismatch")
	}

	return players, nil
}

func (r *Rcon) List() (PlayerList, error) {
	resp, err := r.Execute("list")
	if err != nil {
		slog.Error("Failed to send list command to server", slog.Any("error", err))
	}

	return parsePlayerList(resp)
}
