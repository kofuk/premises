package core_test

import (
	"errors"
	"testing"

	"github.com/kofuk/premises/runner/internal/commands/mclauncher/core"
	"github.com/kofuk/premises/runner/internal/env"
	"github.com/kofuk/premises/runner/internal/system"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"
)

var _ = Describe("LauncherCore", func() {
	var (
		ctrl               *gomock.Controller
		executor           *system.MockCommandExecutor
		settingsRepository *core.MockSettingsRepository
		envProvider        *env.MockEnvProvider
		stateRepository    *core.MockStateRepository
		sut                *core.LauncherCore
	)

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())

		executor = system.NewMockCommandExecutor(ctrl)
		settingsRepository = core.NewMockSettingsRepository(ctrl)
		envProvider = env.NewMockEnvProvider(ctrl)
		stateRepository = core.NewMockStateRepository(ctrl)

		sut = core.NewLauncherCore(settingsRepository, envProvider, stateRepository)
		sut.CommandExecutor = executor
	})

	It("should launch successfully", func() {
		settingsRepository.EXPECT().GetServerPath().Return("/usr/bin/true")
		executor.EXPECT().Run(gomock.Any(), "/usr/bin/true", []string{}, gomock.Any()).Times(1).Return(nil)
		envProvider.EXPECT().GetDataPath(gomock.Any()).AnyTimes().Return("/tmp")

		err := sut.Start(GinkgoT().Context())
		Expect(err).ShouldNot(HaveOccurred())
	})

	It("should retry in case of failure", func() {
		settingsRepository.EXPECT().GetServerPath().Return("/usr/bin/false")
		gomock.InOrder(
			executor.EXPECT().Run(gomock.Any(), "/usr/bin/false", []string{}, gomock.Any()).Times(2).Return(errors.New("error")),
			executor.EXPECT().Run(gomock.Any(), "/usr/bin/false", []string{}, gomock.Any()).Return(nil),
		)
		envProvider.EXPECT().GetDataPath(gomock.Any()).AnyTimes().Return("/tmp")

		err := sut.Start(GinkgoT().Context())
		Expect(err).ShouldNot(HaveOccurred())
	})
})

func Test(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "LauncherCore Suite")
}
