package serverproperties_test

import (
	"os"
	"path/filepath"

	"github.com/kofuk/premises/runner/internal/commands/mclauncher/core"
	"github.com/kofuk/premises/runner/internal/commands/mclauncher/middleware/serverproperties"
	"github.com/kofuk/premises/runner/internal/env"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"
)

var _ = Describe("ServerProperties", func() {
	var (
		ctrl               *gomock.Controller
		settingsRepository *core.MockSettingsRepository
		envProvider        *env.MockEnvProvider
		stateRepository    *core.MockStateRepository
		launcher           *core.LauncherCore
		tempDir            string
	)

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		settingsRepository = core.NewMockSettingsRepository(ctrl)
		envProvider = env.NewMockEnvProvider(ctrl)
		stateRepository = core.NewMockStateRepository(ctrl)
		tempDir = GinkgoT().TempDir()

		launcher = core.NewLauncherCore(settingsRepository, envProvider, stateRepository)
		launcher.Use(core.StopMiddleware)

		envProvider.EXPECT().GetDataPath("gamedata/server.properties").AnyTimes().Return(filepath.Join(tempDir, "gamedata/server.properties"))
		os.MkdirAll(filepath.Join(tempDir, "gamedata"), 0755)
	})

	It("should create a server.properties file", func() {
		settingsRepository.EXPECT().GetMotd().Return("motd")
		settingsRepository.EXPECT().GetDifficulty().Return("normal")
		settingsRepository.EXPECT().GetLevelType().Return("default")
		settingsRepository.EXPECT().GetSeed().Return("seed")
		settingsRepository.EXPECT().ServerPropertiesOverrides().Return(map[string]string{
			"key1": "value1",
		})

		sut := serverproperties.NewServerPropertiesMiddleware()
		launcher.Use(sut)

		err := launcher.Start(GinkgoT().Context())
		Expect(err).ToNot(HaveOccurred())

		Expect(filepath.Join(tempDir, "gamedata/server.properties")).To(BeARegularFile())

		content, _ := os.ReadFile(filepath.Join(tempDir, "gamedata/server.properties"))
		Expect(string(content)).To(ContainSubstring("motd=motd"))
		Expect(string(content)).To(ContainSubstring("difficulty=normal"))
		Expect(string(content)).To(ContainSubstring("level-type=default"))
		Expect(string(content)).To(ContainSubstring("seed=seed"))
		Expect(string(content)).To(ContainSubstring("key1=value1"))

	})
})
