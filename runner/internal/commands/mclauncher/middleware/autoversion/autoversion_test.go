package autoversion_test

import (
	_ "embed"
	"os"
	"path/filepath"
	"testing"

	"github.com/kofuk/premises/runner/internal/commands/mclauncher/core"
	"github.com/kofuk/premises/runner/internal/commands/mclauncher/middleware/autoversion"
	"github.com/kofuk/premises/runner/internal/env"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"
)

//go:embed testdata/level.dat
var levelDat []byte

var _ = Describe("AutoVersionMiddleware", func() {
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
		os.MkdirAll(filepath.Join(tempDir, "gamedata/world"), 0o755)
		os.WriteFile(filepath.Join(tempDir, "gamedata/world/level.dat"), levelDat, 0o644)

		ctrl = gomock.NewController(GinkgoT())

		settingsRepository = core.NewMockSettingsRepository(ctrl)
		envProvider = env.NewMockEnvProvider(ctrl)
		stateRepository = core.NewMockStateRepository(ctrl)

		launcher = core.NewLauncherCore(settingsRepository, envProvider, stateRepository)
		launcher.Middleware(core.StopMiddleware)
	})

	It("should detect server version", func() {
		envProvider.EXPECT().GetDataPath("gamedata/world/level.dat").AnyTimes().Return(filepath.Join(tempDir, "gamedata/world/level.dat"))
		settingsRepository.EXPECT().AutoVersionEnabled().Return(true).AnyTimes()
		settingsRepository.EXPECT().SetMinecraftVersion("1.20.4").Times(1)

		sut := &autoversion.AutoVersionMiddleware{}

		launcher.Middleware(sut)

		err := launcher.Start(GinkgoT().Context())
		Expect(err).ShouldNot(HaveOccurred())
	})

	It("should not override version if AutoVersion is disabled", func() {
		settingsRepository.EXPECT().AutoVersionEnabled().Return(false).AnyTimes()
		settingsRepository.EXPECT().SetMinecraftVersion(gomock.Any()).Times(0)

		sut := &autoversion.AutoVersionMiddleware{}

		launcher.Middleware(sut)

		err := launcher.Start(GinkgoT().Context())
		Expect(err).ShouldNot(HaveOccurred())
	})
})

func Test(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "AutoVersion Suite")
}
