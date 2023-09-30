package proc

import (
	"log"
	"math/rand"
	"os"
	"os/exec"
	"time"
)

type RestartPolicy int

const (
	RestartAlways = iota
	RestartOnFailure
	RestartNever
)

type Task struct {
	name         string
	args         []string
	restart      RestartPolicy
	restartDelay *time.Duration
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

func NewProc(name string, options ...Option) *Task {
	proc := &Task{
		name: name,
	}

	for _, opt := range options {
		opt(proc)
	}

	return proc
}

func (p *Task) waitRestartDelay() {
	if p.restartDelay == nil {
		time.Sleep(time.Duration(rand.Intn(10)) * time.Second)
	} else {
		time.Sleep(*p.restartDelay)
	}
}

func (p *Task) Start() {
L:
	for {
		cmd := exec.Command(p.name, p.args...)
		cmd.Dir = "/"
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		failure := false
		if err := cmd.Run(); err != nil {
			log.Printf("%s: %v", p.name, err)
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
