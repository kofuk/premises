package system

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

func Cmd(ctx context.Context, path string, args []string, options ...CmdOption) error {
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

func CmdOutput(ctx context.Context, path string, args []string, options ...CmdOption) (string, error) {
	tracer := trace.SpanFromContext(ctx).TracerProvider().Tracer(ScopeName)
	ctx, span := tracer.Start(ctx, fmt.Sprintf("EXEC %s", path))
	defer span.End()

	buf := new(strings.Builder)

	cmd := exec.CommandContext(ctx, path, args...)
	cmd.Stdout = buf
	for _, opt := range options {
		opt(cmd)
	}

	span.SetAttributes(
		attribute.String("command.name", path),
		attribute.StringSlice("command.args", cmd.Args),
		attribute.StringSlice("command.env", cmd.Environ()),
	)

	if err := cmd.Run(); err != nil {
		return "", err
	}
	return buf.String(), nil
}

func AptGet(ctx context.Context, args ...string) error {
	if err := Cmd(ctx, "apt-get", args, WithEnv("DEBIAN_FRONTEND=noninteractive")); err == nil {
		return nil
	}
	Cmd(ctx, "dpkg", []string{"--configure", "-a"}, WithEnv("DEBIAN_FRONTEND=noninteractive"))
	return Cmd(ctx, "apt-get", args, WithEnv("DEBIAN_FRONTEND=noninteractive"))
}
