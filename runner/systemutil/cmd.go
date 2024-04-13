package systemutil

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"strings"
	"sync/atomic"
)

var logNum uint64

func createLog() (io.Writer, string, error) {
	logPath := fmt.Sprintf("/tmp/command-%s-%d.log", os.Getenv("PREMISES_RUNNER_COMMAND"), atomic.AddUint64(&logNum, 1)-1)
	log, err := os.Create(logPath)
	if err != nil {
		slog.Error("Unable to create log file", slog.Any("error", err))
		return io.Discard, "<error>", err
	}
	return log, logPath, nil
}

func Cmd(path string, args []string, envs []string) error {
	log, logPath, err := createLog()
	if err != nil {
		return err
	}
	if closer, ok := log.(io.Closer); ok {
		defer closer.Close()
	}

	slog.Info("Execute system command", slog.String("command", path), slog.Any("args", args), slog.String("command_output", logPath))

	cmd := exec.Command(path, args...)
	cmd.Stdout = log
	cmd.Stderr = log
	cmd.Env = append(cmd.Environ(), envs...)
	if err := cmd.Run(); err != nil {
		slog.Error("Command failed", slog.Any("error", err))
		return err
	}
	return nil
}

func CmdOutput(path string, args []string) (string, error) {
	buf := new(strings.Builder)

	cmd := exec.Command(path, args...)
	cmd.Stdout = buf
	if err := cmd.Run(); err != nil {
		return "", err
	}
	return buf.String(), nil
}

func AptGet(args ...string) error {
	if err := Cmd("apt-get", args, []string{"DEBIAN_FRONTEND=noninteractive"}); err == nil {
		return nil
	}
	Cmd("dpkg", []string{"--configure", "-a"}, []string{"DEBIAN_FRONTEND=noninteractive"})
	return Cmd("apt-get", args, []string{"DEBIAN_FRONTEND=noninteractive"})
}
