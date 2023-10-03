package exterior

import (
	"github.com/kofuk/premises/exteriord/exterior/scheduler"
	"github.com/kofuk/premises/exteriord/proc"
)

type Exterior struct {
	scheduler *scheduler.Scheduler
}

func New() *Exterior {
	return &Exterior{
		scheduler: scheduler.NewScheduler(),
	}
}

func (self *Exterior) RegisterTask(description string, proc proc.Proc, deps ...scheduler.TaskId) scheduler.TaskId {
	task := scheduler.NewTask(func() {
		proc.Start()
	}, description, deps...)
	self.scheduler.RegisterTasks(task)
	return task.Id()
}

func (self *Exterior) Run() {
	self.scheduler.Run()
	<-make(chan struct{})
}
