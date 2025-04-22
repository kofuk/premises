package eula_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/kofuk/premises/runner/internal/commands/mclauncher/core"
	"github.com/kofuk/premises/runner/internal/commands/mclauncher/middleware/eula"
	"github.com/kofuk/premises/runner/internal/env"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"
)

var _ = Describe("EulaMiddleware", func() {
	var (
		tempDir            string
		ctrl               *gomock.Controller
		settingsRepository *core.MockSettingsRepository
		envProvider        *env.MockEnvProvider
		stateRepository    *core.MockStateRepository
		launcher           *core.LauncherCore
	)

	BeforeEach(func() {
		tempDir = GinkgoT().TempDir()
		ctrl = gomock.NewController(GinkgoT())
		settingsRepository = core.NewMockSettingsRepository(ctrl)
		envProvider = env.NewMockEnvProvider(ctrl)
		stateRepository = core.NewMockStateRepository(ctrl)

		launcher = core.NewLauncherCore(settingsRepository, envProvider, stateRepository)
		launcher.Use(core.StopMiddleware)
	})

	It("should sign to EULA file", func() {
		envProvider.EXPECT().GetDataPath(gomock.Any()).AnyTimes().Return(tempDir)

		sut := eula.NewEulaMiddleware()

		launcher.Use(sut)

		err := launcher.Start(GinkgoT().Context())
		Expect(err).ShouldNot(HaveOccurred())

		eulaFilePath := filepath.Join(tempDir, "eula.txt")
		Expect(eulaFilePath).To(BeAnExistingFile())

		content, _ := os.ReadFile(eulaFilePath)
		Expect(string(content)).To(ContainSubstring("eula=true"))
	})
})

func Test(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "EulaMiddleware Suite")
}
