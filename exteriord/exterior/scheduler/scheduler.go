package scheduler

import (
	"log"

	"github.com/google/uuid"
	"golang.org/x/exp/slices"
)

type TaskId string

type Task struct {
	taskId      TaskId
	description string
	fn          func()
	deps        []TaskId
	started     bool
}

type Scheduler struct {
	tasks map[TaskId]*Task
}

func (self *Task) Id() TaskId {
	return self.taskId
}

func NewTask(fn func(), description string, deps ...TaskId) *Task {
	taskId := TaskId(uuid.New().String())
	task := Task{
		taskId: taskId,
		fn:     fn,
	}

	for _, dep := range deps {
		task.deps = append(task.deps, dep)
	}

	return &task
}

func NewScheduler() *Scheduler {
	return &Scheduler{
		tasks: make(map[TaskId]*Task),
	}
}

func (self *Task) runTask(notifyComplete chan TaskId) {
	self.fn()
	notifyComplete <- self.taskId
}

func (self *Scheduler) RegisterTasks(tasks ...*Task) {
	for _, task := range tasks {
		self.tasks[task.taskId] = task
	}
}

func (self *Scheduler) Run() {
	notifyComplete := make(chan TaskId)

	completedTasks := 0

	for {
		for _, task := range self.tasks {
			if !task.started && len(task.deps) == 0 {
				log.Println("Starting", task.description)

				go task.runTask(notifyComplete)

				task.started = true
			}
		}

		if completedTasks == len(self.tasks) {
			return
		}

		select {
		case taskId := <-notifyComplete:
			completedTasks++

			for _, dep := range self.tasks {
				dep.deps = slices.DeleteFunc(dep.deps, func(e TaskId) bool {
					return e == taskId
				})
			}
		}
	}
}
