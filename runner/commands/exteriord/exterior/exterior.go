package exterior

import (
	"github.com/kofuk/premises/runner/commands/exteriord/exterior/scheduler"
	"github.com/kofuk/premises/runner/commands/exteriord/proc"
)

type Exterior struct {
	scheduler *scheduler.Scheduler
}

func New() *Exterior {
	return &Exterior{
		scheduler: scheduler.NewScheduler(),
	}
}

func (e *Exterior) RegisterTask(description string, proc proc.Proc, deps ...scheduler.TaskID) scheduler.TaskID {
	task := scheduler.NewTask(func() {
		proc.Start()
	}, description, deps...)
	e.scheduler.RegisterTasks(task)
	return task.ID()
}

func (e *Exterior) Run() {
	e.scheduler.Run()
	<-make(chan struct{})
}
