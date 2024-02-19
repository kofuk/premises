package systemutil

import (
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"sync/atomic"
)

var logNum uint64

func Cmd(cmdPath string, args []string, envs []string) error {
	logPath := fmt.Sprintf("/tmp/command-%d.log", atomic.AddUint64(&logNum, 1)-1)
	log, err := os.Create(logPath)
	if err != nil {
		slog.Error("Unable to create log file", slog.Any("error", err))
		return err
	}
	defer log.Close()

	slog.Info("Execute system command", slog.String("command", cmdPath), slog.Any("args", args), slog.String("command_output", logPath))

	cmd := exec.Command(cmdPath, args...)
	cmd.Stdout = log
	cmd.Stderr = log
	cmd.Env = append(cmd.Environ(), envs...)
	if err := cmd.Run(); err != nil {
		slog.Error("Command failed", slog.Any("error", err))
		return err
	}
	return nil
}

func AptGet(args ...string) error {
	if err := Cmd("apt-get", args, []string{"DEBIAN_FRONTEND=noninteractive"}); err == nil {
		return nil
	}
	Cmd("dpkg", []string{"--configure", "-a"}, []string{"DEBIAN_FRONTEND=noninteractive"})
	return Cmd("apt-get", args, []string{"DEBIAN_FRONTEND=noninteractive"})
}
