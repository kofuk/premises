package watchdog_test

import (
	"github.com/kofuk/premises/runner/internal/commands/mclauncher/core"
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
		lc       *core.MockLauncherContext
	)

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		executor = rcon.NewMockRconExecutorInterface(ctrl)
		rc = rcon.NewRcon(executor)
		lc = core.NewMockLauncherContext(ctrl)
		lc.EXPECT().Context().AnyTimes().Return(GinkgoT().Context())
	})

	It("should run one time initialization when server goes online", func() {
		executor.EXPECT().Exec("op user1").Times(1).Return("", nil)
		executor.EXPECT().Exec("whitelist add user2").Times(1).Return("", nil)
		executor.EXPECT().Exec("seed").Times(1).Return("Seed: [5947924885426060132]", nil)

		settingsRepository := core.NewMockSettingsRepository(ctrl)
		settingsRepository.EXPECT().GetWorldName().Return("foo")
		settingsRepository.EXPECT().GetMinecraftVersion().Return("1.16.5")

		lc.EXPECT().Settings().AnyTimes().Return(settingsRepository)

		wd := watchdog.NewOneTimeInitWatchdog(rc, []string{"user1"}, []string{"user2"})

		status := &watchdog.Status{
			Online: true,
		}

		err := wd.Check(lc, 0, status)
		Expect(err).To(BeNil())

		err = wd.Check(lc, 1, status)
		Expect(err).To(BeNil())
	})
})
