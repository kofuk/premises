package scheduler_test

import (
	"sync/atomic"
	"testing"

	"github.com/kofuk/premises/runner/internal/commands/exteriord/exterior/scheduler"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Scheduler", func() {
	It("should run tasks in the correct order", func() {
		num := atomic.Int32{}

		task1 := scheduler.NewTask(func() {
			num.Add(1)
			Expect(int(num.Load())).To(Equal(1))
		}, "Task 1")
		task21 := scheduler.NewTask(func() {
			num.Add(1)
			curNum := int(num.Load())
			Expect(curNum).To(Or(Equal(2), Equal(3)))
		}, "Task 2-1", task1.ID())
		task22 := scheduler.NewTask(func() {
			num.Add(1)
			curNum := int(num.Load())
			Expect(curNum).To(Or(Equal(2), Equal(3)))
		}, "Task 2-2", task1.ID())
		task3 := scheduler.NewTask(func() {
			num.Add(1)
			Expect(int(num.Load())).To(Equal(4))
		}, "Task 3", task21.ID(), task22.ID())

		scheduler := scheduler.NewScheduler()
		scheduler.RegisterTasks(task1, task21, task22, task3)
		scheduler.Run()
	})
})

func Test(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Scheduler Suite")
}
