package watchdog_test

import (
	"github.com/kofuk/premises/runner/internal/commands/mclauncher/middleware/monitoring/watchdog"
	"github.com/kofuk/premises/runner/internal/commands/mclauncher/rcon"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"
)

var _ = Describe("OneTimeInitWatchdog", func() {
	var (
		ctrl     *gomock.Controller
		executor *rcon.MockRconExecutorInterface
		rc       *rcon.Rcon
	)

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		executor = rcon.NewMockRconExecutorInterface(ctrl)
		rc = rcon.NewRcon(executor)
	})

	It("should add ops and whitelist when server goes online", func() {
		executor.EXPECT().Exec("op user1").Times(1).Return("", nil)
		executor.EXPECT().Exec("whitelist add user2").Times(1).Return("", nil)

		wd := watchdog.NewOneTimeInitWatchdog(rc, []string{"user1"}, []string{"user2"})

		status := &watchdog.Status{
			Online: true,
		}

		err := wd.Check(GinkgoT().Context(), 0, status)

		Expect(err).To(BeNil())
	})
})
