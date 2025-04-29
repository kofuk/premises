package watchdog_test

import (
	"github.com/kofuk/premises/runner/internal/commands/mclauncher/core"
	"github.com/kofuk/premises/runner/internal/commands/mclauncher/middleware/monitoring/watchdog"
	"github.com/kofuk/premises/runner/internal/commands/mclauncher/rcon"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"
)

var _ = Describe("ActivenessWatchdog", func() {
	var (
		ctrl     *gomock.Controller
		executor *rcon.MockRconExecutorInterface
		rc       *rcon.Rcon
		lc       *core.MockLauncherContext
	)

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		executor = rcon.NewMockRconExecutorInterface(ctrl)
		rc = rcon.NewRcon(executor)
		lc = core.NewMockLauncherContext(ctrl)
		lc.EXPECT().Context().AnyTimes().Return(GinkgoT().Context())
	})

	It("should stop the server after timeout", func() {
		gomock.InOrder(
			executor.EXPECT().Exec("list").Return("There are 1 of a max of 20 players online: kofun8", nil), // 0
			executor.EXPECT().Exec("list").Return("There are 0 of a max of 20 players online: ", nil),       // 60
			executor.EXPECT().Exec("list").Return("There are 0 of a max of 20 players online: ", nil),       // 120
			executor.EXPECT().Exec("stop").Return("", nil),                                                  // 120
		)

		wd := watchdog.NewActivenessWatchdog(rc, 1)
		status := &watchdog.Status{
			Online: true,
		}

		for _, time := range []int{0, 60, 120} {
			err := wd.Check(lc, time, status)
			Expect(err).To(BeNil())
		}
	})

	It("should calculate correct timeout even if users login/logout the server", func() {
		gomock.InOrder(
			executor.EXPECT().Exec("list").Return("There are 1 of a max of 20 players online: kofun8", nil), // 0
			executor.EXPECT().Exec("list").Return("There are 0 of a max of 20 players online: ", nil),       // 60
			executor.EXPECT().Exec("list").Return("There are 1 of a max of 20 players online: kofun8", nil), // 120
			executor.EXPECT().Exec("list").Return("There are 0 of a max of 20 players online: ", nil),       // 180
			executor.EXPECT().Exec("list").Return("There are 0 of a max of 20 players online: ", nil),       // 240
			executor.EXPECT().Exec("stop").Return("", nil),                                                  // 240
		)

		wd := watchdog.NewActivenessWatchdog(rc, 1)
		status := &watchdog.Status{
			Online: true,
		}

		for _, time := range []int{0, 60, 120, 180, 240} {
			err := wd.Check(lc, time, status)
			Expect(err).To(BeNil())
		}
	})

	It("should start counting when server goes online", func() {
		gomock.InOrder(
			executor.EXPECT().Exec("list").Return("There are 0 of a max of 20 players online: ", nil),       // 180
			executor.EXPECT().Exec("list").Return("There are 1 of a max of 20 players online: kofun8", nil), // 240
			executor.EXPECT().Exec("list").Return("There are 0 of a max of 20 players online: ", nil),       // 300
			executor.EXPECT().Exec("list").Return("There are 0 of a max of 20 players online: ", nil),       // 360
			executor.EXPECT().Exec("stop").Return("", nil),                                                  // 360
		)

		wd := watchdog.NewActivenessWatchdog(rc, 1)

		calls := []struct {
			watchID int
			online  bool
		}{
			{watchID: 0, online: false},
			{watchID: 60, online: false},
			{watchID: 120, online: false},
			{watchID: 121, online: true},
			{watchID: 180, online: true},
			{watchID: 240, online: true},
			{watchID: 300, online: true},
			{watchID: 360, online: true},
		}

		for _, call := range calls {
			status := &watchdog.Status{
				Online: call.online,
			}
			err := wd.Check(lc, call.watchID, status)
			Expect(err).To(BeNil())
		}
	})
})
