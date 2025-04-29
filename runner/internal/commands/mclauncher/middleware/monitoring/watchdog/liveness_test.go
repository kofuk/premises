package watchdog_test

import (
	"net"

	"github.com/kofuk/premises/runner/internal/commands/mclauncher/core"
	"github.com/kofuk/premises/runner/internal/commands/mclauncher/middleware/monitoring/watchdog"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"
)

var _ = Describe("LivenessWatchdog", func() {
	var (
		ctrl *gomock.Controller
		lc *core.MockLauncherContext
	)

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())

		lc = core.NewMockLauncherContext(ctrl)
		lc.EXPECT().Context().AnyTimes().Return(GinkgoT().Context())
	})

	It("should return the correct name", func() {
		watchdog := watchdog.NewLivenessWatchdog()
		Expect(watchdog.Name()).To(Equal("LivenessWatchdog"))
	})

	It("should check the liveness of the server", func() {
		listener, err := net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			Fail("Failed to start TCP listener")
		}
		addr := listener.Addr().String()
		// Release the port here to get a free one
		listener.Close()

		wd := watchdog.NewLivenessWatchdog(addr)
		status := &watchdog.Status{}
		err = wd.Check(lc, 0, status)
		Expect(err).To(BeNil())
		Expect(status.Online).To(BeFalse())
	})

	It("should check the liveness of the server when online", func() {
		listener, err := net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			Fail("Failed to start TCP listener")
		}
		defer listener.Close()

		go func() {
			for {
				conn, err := listener.Accept()
				if err != nil {
					return
				}
				conn.Close()
			}
		}()

		wd := watchdog.NewLivenessWatchdog(listener.Addr().String())
		status := &watchdog.Status{}
		err = wd.Check(lc, 0, status)
		Expect(err).To(BeNil())
		Expect(status.Online).To(BeTrue())
	})
})
