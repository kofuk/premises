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

func (t *Task) ID() TaskID {
	return t.taskID
}

func NewTask(fn func(), description string, deps ...TaskID) *Task {
	taskID := TaskID(uuid.New().String())
	task := Task{
		taskID:      taskID,
		description: description,
		fn:          fn,
	}

	task.deps = append(task.deps, deps...)

	return &task
}

func NewScheduler() *Scheduler {
	return &Scheduler{
		tasks:         make(map[TaskID]*Task),
		competedTasks: make(map[TaskID]struct{}),
	}
}

func (t *Task) runTask(notifyComplete chan TaskID) {
	t.fn()
	notifyComplete <- t.taskID
}

func (s *Scheduler) RegisterTasks(tasks ...*Task) {
	for _, task := range tasks {
		s.tasks[task.taskID] = task
	}
}

func (s *Scheduler) Run() {
	notifyComplete := make(chan TaskID)

	completedTasks := 0

	for completedTasks != len(s.tasks) {
		for _, task := range s.tasks {
			uncompletedDepsCount := 0
			for _, depID := range task.deps {
				if _, ok := s.competedTasks[depID]; !ok {
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

		s.competedTasks[taskID] = struct{}{}
	}
}
