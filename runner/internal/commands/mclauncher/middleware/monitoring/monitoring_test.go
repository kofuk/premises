package monitoring_test

import (
	"context"
	"testing"
	"time"

	"github.com/kofuk/premises/runner/internal/commands/mclauncher/core"
	"github.com/kofuk/premises/runner/internal/commands/mclauncher/middleware/monitoring"
	"github.com/kofuk/premises/runner/internal/commands/mclauncher/middleware/monitoring/watchdog"
	"github.com/kofuk/premises/runner/internal/env"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"
)

type sleepMiddleware struct {
	duration time.Duration
}

var _ core.Middleware = (*sleepMiddleware)(nil)

func (m *sleepMiddleware) Wrap(next core.HandlerFunc) core.HandlerFunc {
	return func(c *core.LauncherContext) error {
		time.Sleep(m.duration)
		return next(c)
	}
}

var _ = Describe("Monitoring Middleware", func() {
	var (
		ctrl               *gomock.Controller
		settingsRepository *core.MockSettingsRepository
		envProvider        *env.MockEnvProvider
		stateRepository    *core.MockStateRepository
		launcher           *core.LauncherCore
	)

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		settingsRepository = core.NewMockSettingsRepository(ctrl)
		envProvider = env.NewMockEnvProvider(ctrl)
		stateRepository = core.NewMockStateRepository(ctrl)

		launcher = core.NewLauncherCore(settingsRepository, envProvider, stateRepository)
		launcher.Middleware(core.StopMiddleware)
	})

	It("should trigger watchdogs", func() {
		wd := watchdog.NewMockWatchdog(ctrl)
		gomock.InOrder(
			wd.EXPECT().Check(gomock.Any(), 0, &watchdog.Status{Online: false}).Do(
				func(ctx context.Context, id int, status *watchdog.Status) {
					status.Online = true
				},
			).Return(nil),
			wd.EXPECT().Check(gomock.Any(), 1, &watchdog.Status{Online: false}).Do(
				func(ctx context.Context, id int, status *watchdog.Status) {
					status.Online = true
				},
			).Return(nil),
			wd.EXPECT().Check(gomock.Any(), gomock.Any(), &watchdog.Status{Online: false}).AnyTimes().Return(nil),
		)

		sut := monitoring.NewMonitoringMiddleware()
		sut.AddWatchdog(wd)

		launcher.Middleware(&sleepMiddleware{duration: 3 * time.Second})
		launcher.Middleware(sut)

		err := launcher.Start(GinkgoT().Context())
		Expect(err).NotTo(HaveOccurred())
	})
})

func Test(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "MonitoringMiddleware Suite")
}
