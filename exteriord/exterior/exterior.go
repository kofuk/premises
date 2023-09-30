package exterior

import (
	"github.com/kofuk/premises/exteriord/proc"
)

type Exterior struct {
	tasks []proc.Task
}

func New() *Exterior {
	return &Exterior{}
}

func (self *Exterior) RegisterTask(task proc.Task) {
	self.tasks = append(self.tasks, task)
}

func (self *Exterior) Run() {
	for _, task := range self.tasks {
		go task.Start()
	}
	<-make(chan any)
}
