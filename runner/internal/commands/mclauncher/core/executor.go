package core

import (
	"context"
	"errors"
	"os"
	"os/exec"

	"github.com/kofuk/premises/runner/internal/system"
)

type GameExecutor struct {
	pid int
}

func NewGameExecutor() *GameExecutor {
	return &GameExecutor{}
}

var _ system.CommandExecutor = (*GameExecutor)(nil)

func (e *GameExecutor) Run(ctx context.Context, command string, args []string, opts ...system.CmdOption) error {
	if e.pid != 0 {
		return errors.New("process already running")
	}

	cmd := exec.Command(command, args...)
	for _, opt := range opts {
		opt(cmd)
	}

	e.pid = cmd.Process.Pid

	if err := cmd.Start(); err != nil {
		return err
	}

	err := cmd.Wait()

	e.pid = 0

	return err
}

func (e *GameExecutor) Kill() error {
	if e.pid == 0 {
		return nil
	}

	proc, err := os.FindProcess(e.pid)
	if err != nil {
		return err
	}
	proc.Kill()

	e.pid = 0
	return nil
}
