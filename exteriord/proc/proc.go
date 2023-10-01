package proc

import (
	"log"
	"math/rand"
	"os"
	"os/exec"
	"syscall"
	"time"
)

type RestartPolicy int

const (
	RestartAlways RestartPolicy = iota
	RestartOnFailure
	RestartNever
)

type ExecUserType int

const (
	UserRestricted ExecUserType = iota
	UserPrivileged
)

type ProcType int

const (
	ProcDaemon ProcType = iota
	ProcOneShot
)

type Task struct {
	description  string
	execPath     string
	args         []string
	restart      RestartPolicy
	restartDelay *time.Duration
	userType     ExecUserType
	procType     ProcType
}

type Option func(p *Task)

func Args(args ...string) Option {
	return func(p *Task) {
		p.args = args
	}
}

func Restart(restart RestartPolicy) Option {
	return func(p *Task) {
		p.restart = restart
		if (restart == RestartAlways) && p.procType == ProcOneShot {
			p.procType = ProcDaemon
		}
	}
}

func RestartDelay(d time.Duration) Option {
	return func(p *Task) {
		p.restartDelay = &d
	}
}

func RestartRandomDelay() Option {
	return func(p *Task) {
		p.restartDelay = nil
	}
}

func UserType(userType ExecUserType) Option {
	return func(p *Task) {
		p.userType = userType
	}
}

func Description(description string) Option {
	return func(p *Task) {
		p.description = description
	}
}

func Type(procType ProcType) Option {
	return func(p *Task) {
		p.procType = procType
		if procType == ProcOneShot && p.restart == RestartAlways {
			p.restart = RestartOnFailure
		}
	}
}

func NewProc(execPath string, options ...Option) Task {
	proc := Task{
		description: execPath,
		execPath:    execPath,
	}

	for _, opt := range options {
		opt(&proc)
	}

	return proc
}

func (p Task) waitRestartDelay() {
	if p.restartDelay == nil {
		time.Sleep(time.Duration(rand.Intn(10)) * time.Second)
	} else {
		time.Sleep(*p.restartDelay)
	}
}

func (p Task) startTask() {
L:
	for {
		cmd := exec.Command(p.execPath, p.args...)
		cmd.Dir = "/"
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		if p.userType == UserPrivileged {
			// Do nothing
		} else if p.userType == UserRestricted {
			cmd.SysProcAttr = &syscall.SysProcAttr{
				Credential: &syscall.Credential{
					Uid: 1000,
					Gid: 1000,
				},
			}
		}

		failure := false
		if err := cmd.Run(); err != nil {
			log.Printf("%s: %v", p.execPath, err)
			failure = true
		}

		switch p.restart {
		case RestartAlways:
			p.waitRestartDelay()
			continue L

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

func (p Task) Start() {
	switch p.procType {
	case ProcOneShot:
		p.startTask()

	case ProcDaemon:
		go p.startTask()
	}
}

func (p Task) GetDescription() string {
	return p.description
}
