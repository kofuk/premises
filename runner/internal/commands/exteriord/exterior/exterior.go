package exterior

import (
	"context"

	"github.com/kofuk/premises/runner/internal/commands/exteriord/exterior/scheduler"
	"github.com/kofuk/premises/runner/internal/commands/exteriord/proc"
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

func (e *Exterior) Run(ctx context.Context) {
	e.scheduler.Run()
	<-ctx.Done()
}
