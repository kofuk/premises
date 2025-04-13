package rcon

import (
	"errors"
	"log/slog"
	"regexp"
	"strconv"
	"strings"
)

type ListOutput struct {
	MaxPlayers int
	Players    []string
}

var (
	playerListRegexp = regexp.MustCompile("^There are ([0-9]+) of a max of ([0-9]+) players online: (.*)$")
)

func ParseListOutput(output string) (*ListOutput, error) {
	match := playerListRegexp.FindStringSubmatch(output)
	if match == nil {
		return nil, errors.New("invalid /list output")
	}

	count, _ := strconv.Atoi(match[1]) // Error should not be occurred
	max, _ := strconv.Atoi(match[2])   // Error should not be occurred

	var players []string
	for _, p := range strings.Split(match[3], ", ") {
		if p != "" {
			players = append(players, strings.Trim(p, " "))
		}
	}

	if count != len(players) {
		return nil, errors.New("player count mismatch")
	}

	return &ListOutput{MaxPlayers: max, Players: players}, nil
}

func (r *Rcon) List() (*ListOutput, error) {
	resp, err := r.executor.Exec("list")
	if err != nil {
		slog.Error("Failed to send list command to server", slog.Any("error", err))
	}

	return ParseListOutput(resp)
}
