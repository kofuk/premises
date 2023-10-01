package exterior

import (
	"log"

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
		log.Printf("Starting %s\n", task.GetDescription())
		go task.Start()
	}
	<-make(chan any)
}
