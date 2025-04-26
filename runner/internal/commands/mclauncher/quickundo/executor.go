package quickundo

import (
	"context"
	"errors"
	"os"
	"os/exec"

	"github.com/kofuk/premises/runner/internal/system"
)

type KillableCommandExecutor struct {
	pid int
}

func NewKillableCommandExecutor() *KillableCommandExecutor {
	return &KillableCommandExecutor{}
}

var _ system.CommandExecutor = (*KillableCommandExecutor)(nil)

func (e *KillableCommandExecutor) Run(ctx context.Context, command string, args []string, opts ...system.CmdOption) error {
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

func (e *KillableCommandExecutor) Kill() error {
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
