package proc

import (
	"log/slog"
	"math/rand"
	"os"
	"os/exec"
	"syscall"
	"time"

	"github.com/kofuk/premises/runner/internal/system"
)

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
		if err := cmd.Run(); err != nil {
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
