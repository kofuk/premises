package proc

import (
	"context"
	"fmt"
	"log/slog"
	"math/rand"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"time"

	potel "github.com/kofuk/premises/internal/otel"
	"github.com/kofuk/premises/runner/internal/system"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

const ScopeName = "github.com/kofuk/premises/runner/internal/commands/exteriord/proc"

type RestartPolicy int

const (
	RestartOnFailure RestartPolicy = iota
	RestartNever
)

type ExecUserType int

const (
	UserRestricted ExecUserType = iota
	UserPrivileged
)

type Proc struct {
	execPath     string
	args         []string
	restart      RestartPolicy
	restartDelay *time.Duration
	userType     ExecUserType
}

type Option func(p *Proc)

func Args(args ...string) Option {
	return func(p *Proc) {
		p.args = args
	}
}

func Restart(restart RestartPolicy) Option {
	return func(p *Proc) {
		p.restart = restart
	}
}

func RestartDelay(d time.Duration) Option {
	return func(p *Proc) {
		p.restartDelay = &d
	}
}

func RestartRandomDelay() Option {
	return func(p *Proc) {
		p.restartDelay = nil
	}
}

func UserType(userType ExecUserType) Option {
	return func(p *Proc) {
		p.userType = userType
	}
}

func NewProc(execPath string, options ...Option) Proc {
	proc := Proc{
		execPath: execPath,
	}

	for _, opt := range options {
		opt(&proc)
	}

	return proc
}

func (p Proc) waitRestartDelay() {
	if p.restartDelay == nil {
		time.Sleep(time.Duration(rand.Intn(10)) * time.Second)
	} else {
		time.Sleep(*p.restartDelay)
	}
}

func runCommand(cmd *exec.Cmd) error {
	args := cmd.Args
	if len(args) > 1 {
		args = args[1:]
	}

	path := cmd.Path
	if strings.HasSuffix(path, "premises-runner") {
		if len(args) > 0 {
			path = fmt.Sprintf("RUNNER(%s)", strings.TrimPrefix(args[0], "--"))
		}
	}

	tracer := otel.GetTracerProvider().Tracer(ScopeName)
	ctx, span := tracer.Start(context.Background(), fmt.Sprintf("EXEC %s", path),
		trace.WithNewRoot(),
	)
	defer span.End()
	span.SetAttributes(
		attribute.String("command.name", path),
		attribute.StringSlice("command.args", args),
	)

	cmd.Env = append(cmd.Environ(), fmt.Sprintf("TRACEPARENT=%s", potel.TraceContextFromContext(ctx)))

	slog.Info("Executing process",
		slog.String("command", path),
		slog.Any("args", args),
		slog.String("trace_id", span.SpanContext().TraceID().String()),
	)

	if err := cmd.Run(); err != nil {
		span.SetStatus(codes.Error, err.Error())
		return err
	}

	return nil
}

func (p Proc) Start() {
L:
	for {
		cmd := exec.Command(p.execPath, p.args...)
		cmd.Dir = "/"
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		if p.userType == UserPrivileged {
			// Do nothing
		} else if p.userType == UserRestricted {
			uid, gid, err := system.GetAppUserID()
			if err != nil {
				slog.Error("Error retrieving uid and gid for premises user. Process will be executed with root user")
			}

			cmd.SysProcAttr = &syscall.SysProcAttr{
				Credential: &syscall.Credential{
					Uid: uint32(uid),
					Gid: uint32(gid),
				},
			}
		}

		failure := false
		if err := runCommand(cmd); err != nil {
			slog.Error("Command failed", slog.Any("error", err), slog.String("executable", p.execPath))
			failure = true
		}

		switch p.restart {
		case RestartOnFailure:
			if failure {
				p.waitRestartDelay()
				continue L
			}
			break L

		case RestartNever:
			break L
		}
	}
}
