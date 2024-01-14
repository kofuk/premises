package scheduler

import (
	"log/slog"

	"github.com/google/uuid"
)

type TaskID string

type Task struct {
	taskID      TaskID
	description string
	fn          func()
	deps        []TaskID
	started     bool
}

type Scheduler struct {
	tasks         map[TaskID]*Task
	competedTasks map[TaskID]struct{}
}

func (self *Task) ID() TaskID {
	return self.taskID
}

func NewTask(fn func(), description string, deps ...TaskID) *Task {
	taskID := TaskID(uuid.New().String())
	task := Task{
		taskID:      taskID,
		description: description,
		fn:          fn,
	}

	for _, dep := range deps {
		task.deps = append(task.deps, dep)
	}

	return &task
}

func NewScheduler() *Scheduler {
	return &Scheduler{
		tasks:         make(map[TaskID]*Task),
		competedTasks: make(map[TaskID]struct{}),
	}
}

func (self *Task) runTask(notifyComplete chan TaskID) {
	self.fn()
	notifyComplete <- self.taskID
}

func (self *Scheduler) RegisterTasks(tasks ...*Task) {
	for _, task := range tasks {
		self.tasks[task.taskID] = task
	}
}

func (self *Scheduler) Run() {
	notifyComplete := make(chan TaskID)

	completedTasks := 0

	for completedTasks != len(self.tasks) {
		for _, task := range self.tasks {
			uncompletedDepsCount := 0
			for _, depID := range task.deps {
				if _, ok := self.competedTasks[depID]; !ok {
					uncompletedDepsCount++
				}
			}

			if !task.started && uncompletedDepsCount == 0 {
				slog.Info("Starting task", slog.String("description", task.description), slog.String("id", string(task.taskID)))
				go task.runTask(notifyComplete)
				task.started = true
			}
		}

		taskID := <-notifyComplete
		completedTasks++

		self.competedTasks[taskID] = struct{}{}
	}
}
