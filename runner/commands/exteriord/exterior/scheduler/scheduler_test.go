package scheduler

import (
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_Scheduler_Run(t *testing.T) {
	if testing.Short() {
		t.SkipNow()
	}

	num := atomic.Int32{}

	task1 := NewTask(func() {
		num.Add(1)
		assert.Equal(t, 1, int(num.Load()))
	}, "Task 1")
	task21 := NewTask(func() {
		num.Add(1)
		curNum := int(num.Load())
		assert.True(t, curNum == 2 || curNum == 3)
	}, "Task 2-1", task1.Id())
	task22 := NewTask(func() {
		num.Add(1)
		curNum := int(num.Load())
		assert.True(t, curNum == 2 || curNum == 3)
	}, "Task 2-2", task1.Id())
	task3 := NewTask(func() {
		num.Add(1)
		assert.Equal(t, 4, int(num.Load()))
	}, "Task 3", task21.Id(), task22.Id())

	scheduler := NewScheduler()
	scheduler.RegisterTasks(task1, task21, task22, task3)
	scheduler.Run()
}
