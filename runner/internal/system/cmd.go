package system

//go:generate go run go.uber.org/mock/mockgen@v0.5.0 -destination cmd_mock.go -package system . CommandExecutor

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync/atomic"

	"github.com/kofuk/premises/runner/internal/env"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

const ScopeName = "github.com/kofuk/premises/runner/internal/system"

var logNum uint64

type CommandExecutor interface {
	Run(ctx context.Context, path string, args []string, options ...CmdOption) error
}

type SimpleExecutor struct{}

var DefaultExecutor CommandExecutor = new(SimpleExecutor)

func createLog() (io.Writer, string, error) {
	for {
		logPath := filepath.Join(env.GetTempDir(), fmt.Sprintf("command-%d.log", atomic.AddUint64(&logNum, 1)-1))
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

func WithOutput(w io.Writer) CmdOption {
	return func(cmd *exec.Cmd) {
		cmd.Stdout = w
	}
}

func (e *SimpleExecutor) Run(ctx context.Context, path string, args []string, options ...CmdOption) error {
	tracer := trace.SpanFromContext(ctx).TracerProvider().Tracer(ScopeName)
	ctx, span := tracer.Start(ctx, fmt.Sprintf("EXEC %s", path))
	defer span.End()

	log, logPath, err := createLog()
	if err != nil {
		return err
	}
	if closer, ok := log.(io.Closer); ok {
		defer closer.Close()
	}

	slog.Info("Execute system command", slog.String("command", path), slog.Any("args", args), slog.String("command_output", logPath))

	cmd := exec.CommandContext(ctx, path, args...)
	cmd.Stdout = log
	cmd.Stderr = log
	cmd.Env = cmd.Environ()
	for _, opt := range options {
		opt(cmd)
	}

	span.SetAttributes(
		attribute.String("command.name", path),
		attribute.StringSlice("command.args", cmd.Args),
		attribute.StringSlice("command.env", cmd.Environ()),
	)

	if err := cmd.Run(); err != nil {
		span.SetStatus(codes.Error, err.Error())
		slog.Error("Command failed", slog.Any("error", err))
		return err
	}
	return nil
}

func RunWithOutput(ctx context.Context, executor CommandExecutor, path string, args []string, options ...CmdOption) (string, error) {
	output := new(strings.Builder)
	err := executor.Run(ctx, path, args, append(options, WithOutput(output))...)
	return output.String(), err
}

func AptGet(ctx context.Context, args ...string) error {
	if err := DefaultExecutor.Run(ctx, "apt-get", args, WithEnv("DEBIAN_FRONTEND=noninteractive")); err == nil {
		return nil
	}
	DefaultExecutor.Run(ctx, "dpkg", []string{"--configure", "-a"}, WithEnv("DEBIAN_FRONTEND=noninteractive"))
	return DefaultExecutor.Run(ctx, "apt-get", args, WithEnv("DEBIAN_FRONTEND=noninteractive"))
}
