package system

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync/atomic"

	"github.com/kofuk/premises/runner/fs"
)

var logNum uint64

func createLog() (io.Writer, string, error) {
	for {
		logPath := filepath.Join(fs.GetTempDir(), fmt.Sprintf("command-%d.log", atomic.AddUint64(&logNum, 1)-1))
		log, err := os.OpenFile(logPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644)
		if err != nil {
			if os.IsExist(err) {
				continue
			}
			slog.Error("Unable to create log file", slog.Any("error", err))
			return io.Discard, "<error>", err
		}
		return log, logPath, nil
	}
}

type CmdOption func(cmd *exec.Cmd)

func WithEnv(env string) CmdOption {
	return func(cmd *exec.Cmd) {
		cmd.Env = append(cmd.Env, env)
	}
}

func WithWorkingDir(dir string) CmdOption {
	return func(cmd *exec.Cmd) {
		cmd.Dir = dir
	}
}

func Cmd(path string, args []string, options ...CmdOption) error {
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
	cmd.Env = cmd.Environ()
	for _, opt := range options {
		opt(cmd)
	}

	if err := cmd.Run(); err != nil {
		slog.Error("Command failed", slog.Any("error", err))
		return err
	}
	return nil
}

func CmdOutput(path string, args []string, options ...CmdOption) (string, error) {
	buf := new(strings.Builder)

	cmd := exec.Command(path, args...)
	cmd.Stdout = buf
	for _, opt := range options {
		opt(cmd)
	}

	if err := cmd.Run(); err != nil {
		return "", err
	}
	return buf.String(), nil
}

func AptGet(args ...string) error {
	if err := Cmd("apt-get", args, WithEnv("DEBIAN_FRONTEND=noninteractive")); err == nil {
		return nil
	}
	Cmd("dpkg", []string{"--configure", "-a"}, WithEnv("DEBIAN_FRONTEND=noninteractive"))
	return Cmd("apt-get", args, WithEnv("DEBIAN_FRONTEND=noninteractive"))
}
