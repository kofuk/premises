package world_test

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/kofuk/premises/runner/internal/commands/mclauncher/core"
	"github.com/kofuk/premises/runner/internal/commands/mclauncher/middleware/world"
	"github.com/kofuk/premises/runner/internal/commands/mclauncher/middleware/world/service"
	"github.com/kofuk/premises/runner/internal/env"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"
)

type ErrorMiddleware struct {
	err error
}

var _ core.Middleware = (*ErrorMiddleware)(nil)

func (e *ErrorMiddleware) Wrap(next core.HandlerFunc) core.HandlerFunc {
	return func(c core.LauncherContext) error {
		return e.err
	}
}

var _ = Describe("WorldMiddleware", func() {
	var (
		tempDir            string
		ctrl               *gomock.Controller
		settingsRepository *core.MockSettingsRepository
		envProvider        *env.MockEnvProvider
		stateRepository    *core.MockStateRepository
		worldService       *service.MockWorldServiceInterface
		launcher           *core.LauncherCore
	)

	BeforeEach(func() {
		tempDir = GinkgoT().TempDir()
		os.MkdirAll(filepath.Join(tempDir, "gamedata/world"), 0o755)
		ctrl = gomock.NewController(GinkgoT())

		settingsRepository = core.NewMockSettingsRepository(ctrl)
		settingsRepository.EXPECT().GetWorldName().AnyTimes().Return("foo")

		envProvider = env.NewMockEnvProvider(ctrl)
		stateRepository = core.NewMockStateRepository(ctrl)
		worldService = service.NewMockWorldServiceInterface(ctrl)

		launcher = core.NewLauncherCore(settingsRepository, envProvider, stateRepository)
		launcher.Use(core.StopMiddleware)
	})

	It("should download world if no state stored", func() {
		settingsRepository.EXPECT().GetWorldResourceID().AnyTimes().Return("res-id-1")
		settingsRepository.EXPECT().IsNewWorld().Return(false)
		settingsRepository.EXPECT().SetWorldResourceID("res-id-1")
		gomock.InOrder(
			stateRepository.EXPECT().GetState(gomock.Any(), world.StateKeyWorldKey).Return("", nil), // return empty string here
			stateRepository.EXPECT().RemoveState(gomock.Any(), world.StateKeyWorldKey),
			stateRepository.EXPECT().SetState(gomock.Any(), world.StateKeyWorldKey, "res-id-2"),
		)
		envProvider.EXPECT().GetDataPath("gamedata/world").Return(filepath.Join(tempDir, "gamedata/world"))

		os.WriteFile(filepath.Join(tempDir, "gamedata/world/levdel.dat"), []byte("level"), 0o644)

		worldService.EXPECT().DownloadWorld(gomock.Any(), "res-id-1", gomock.Any()).Return(nil)
		worldService.EXPECT().UploadWorld(gomock.Any(), "foo", gomock.Any()).Return("res-id-2", nil)

		sut := world.NewWorldMiddleware(worldService)

		launcher.Use(sut)

		err := launcher.Start(GinkgoT().Context())
		Expect(err).ShouldNot(HaveOccurred())

		Expect(filepath.Join(tempDir, "gamedata/world/levdel.dat")).NotTo(BeAnExistingFile())
	})

	It("should download world if world is changed", func() {
		settingsRepository.EXPECT().GetWorldResourceID().AnyTimes().Return("res-id-2")
		settingsRepository.EXPECT().IsNewWorld().Return(false)
		settingsRepository.EXPECT().SetWorldResourceID("res-id-2")
		gomock.InOrder(
			stateRepository.EXPECT().GetState(gomock.Any(), world.StateKeyWorldKey).Return("res-id-1", nil),
			stateRepository.EXPECT().RemoveState(gomock.Any(), world.StateKeyWorldKey),
			stateRepository.EXPECT().SetState(gomock.Any(), world.StateKeyWorldKey, "res-id-3"),
		)
		envProvider.EXPECT().GetDataPath("gamedata/world").Return(filepath.Join(tempDir, "gamedata/world"))

		os.WriteFile(filepath.Join(tempDir, "gamedata/world/levdel.dat"), []byte("level"), 0o644)

		worldService.EXPECT().DownloadWorld(gomock.Any(), "res-id-2", gomock.Any()).Return(nil)
		worldService.EXPECT().UploadWorld(gomock.Any(), "foo", gomock.Any()).Return("res-id-3", nil)

		sut := world.NewWorldMiddleware(worldService)

		launcher.Use(sut)

		err := launcher.Start(GinkgoT().Context())
		Expect(err).ShouldNot(HaveOccurred())

		Expect(filepath.Join(tempDir, "gamedata/world/levdel.dat")).NotTo(BeAnExistingFile())
	})

	It("should not download world if world is not changed", func() {
		settingsRepository.EXPECT().GetWorldResourceID().AnyTimes().Return("res-id-1")
		settingsRepository.EXPECT().IsNewWorld().Return(false)
		settingsRepository.EXPECT().SetWorldResourceID("res-id-1")
		gomock.InOrder(
			stateRepository.EXPECT().GetState(gomock.Any(), world.StateKeyWorldKey).Return("res-id-1", nil),
			stateRepository.EXPECT().RemoveState(gomock.Any(), world.StateKeyWorldKey),
			stateRepository.EXPECT().SetState(gomock.Any(), world.StateKeyWorldKey, "res-id-2"),
		)

		os.WriteFile(filepath.Join(tempDir, "gamedata/world/levdel.dat"), []byte("level"), 0o644)

		worldService.EXPECT().UploadWorld(gomock.Any(), "foo", gomock.Any()).Return("res-id-2", nil)

		sut := world.NewWorldMiddleware(worldService)

		launcher.Use(sut)

		err := launcher.Start(GinkgoT().Context())
		Expect(err).ShouldNot(HaveOccurred())

		Expect(filepath.Join(tempDir, "gamedata/world/levdel.dat")).To(BeAnExistingFile())
	})

	It("should update to actual resource ID if latest world is specified", func() {
		settingsRepository.EXPECT().GetWorldResourceID().AnyTimes().Return(world.LatestResourceID)
		settingsRepository.EXPECT().IsNewWorld().Return(false)
		settingsRepository.EXPECT().SetWorldResourceID("res-id-2")
		gomock.InOrder(
			stateRepository.EXPECT().GetState(gomock.Any(), world.StateKeyWorldKey).Return("res-id-1", nil),
			stateRepository.EXPECT().RemoveState(gomock.Any(), world.StateKeyWorldKey),
			stateRepository.EXPECT().SetState(gomock.Any(), world.StateKeyWorldKey, "res-id-3"),
		)
		envProvider.EXPECT().GetDataPath("gamedata/world").Return(filepath.Join(tempDir, "gamedata/world"))

		os.WriteFile(filepath.Join(tempDir, "gamedata/world/levdel.dat"), []byte("level"), 0o644)

		gomock.InOrder(
			worldService.EXPECT().GetLatestResourceID(gomock.Any(), "foo").Return("res-id-2", nil),
			worldService.EXPECT().DownloadWorld(gomock.Any(), "res-id-2", gomock.Any()).Return(nil),
			worldService.EXPECT().UploadWorld(gomock.Any(), "foo", gomock.Any()).Return("res-id-3", nil),
		)

		sut := world.NewWorldMiddleware(worldService)

		launcher.Use(sut)

		err := launcher.Start(GinkgoT().Context())
		Expect(err).ShouldNot(HaveOccurred())

		Expect(filepath.Join(tempDir, "gamedata/world/levdel.dat")).NotTo(BeAnExistingFile())
	})

	It("should not download world if intended to generate a new world", func() {
		settingsRepository.EXPECT().IsNewWorld().Return(true)
		gomock.InOrder(
			stateRepository.EXPECT().RemoveState(gomock.Any(), world.StateKeyWorldKey),
			stateRepository.EXPECT().SetState(gomock.Any(), world.StateKeyWorldKey, "res-id-1"),
		)
		envProvider.EXPECT().GetDataPath("gamedata/world").Return(filepath.Join(tempDir, "gamedata/world"))

		os.WriteFile(filepath.Join(tempDir, "gamedata/world/levdel.dat"), []byte("level"), 0o644)

		worldService.EXPECT().UploadWorld(gomock.Any(), "foo", gomock.Any()).Return("res-id-1", nil)

		sut := world.NewWorldMiddleware(worldService)

		launcher.Use(sut)

		err := launcher.Start(GinkgoT().Context())
		Expect(err).ShouldNot(HaveOccurred())

		Expect(filepath.Join(tempDir, "gamedata/world/levdel.dat")).NotTo(BeAnExistingFile())
	})

	It("should not upload world if unknown error occurred", func() {
		settingsRepository.EXPECT().IsNewWorld().Return(true)
		stateRepository.EXPECT().RemoveState(gomock.Any(), world.StateKeyWorldKey)
		envProvider.EXPECT().GetDataPath("gamedata/world").Return(filepath.Join(tempDir, "gamedata/world"))

		os.WriteFile(filepath.Join(tempDir, "gamedata/world/levdel.dat"), []byte("level"), 0o644)

		sut := world.NewWorldMiddleware(worldService)

		launcher.Use(&ErrorMiddleware{
			err: errors.New("test"),
		})
		launcher.Use(sut)

		err := launcher.Start(GinkgoT().Context())
		Expect(err).To(HaveOccurred())

		Expect(filepath.Join(tempDir, "gamedata/world/levdel.dat")).NotTo(BeAnExistingFile())
	})

	It("should upload world if error is ErrRestart", func() {
		settingsRepository.EXPECT().IsNewWorld().Return(true)
		gomock.InOrder(
			stateRepository.EXPECT().RemoveState(gomock.Any(), world.StateKeyWorldKey),
			stateRepository.EXPECT().SetState(gomock.Any(), world.StateKeyWorldKey, "res-id-1"),
		)
		envProvider.EXPECT().GetDataPath("gamedata/world").Return(filepath.Join(tempDir, "gamedata/world"))

		os.WriteFile(filepath.Join(tempDir, "gamedata/world/levdel.dat"), []byte("level"), 0o644)

		worldService.EXPECT().UploadWorld(gomock.Any(), "foo", gomock.Any()).Return("res-id-1", nil)

		sut := world.NewWorldMiddleware(worldService)

		launcher.Use(&ErrorMiddleware{
			err: core.ErrRestart,
		})
		launcher.Use(sut)

		err := launcher.Start(GinkgoT().Context())
		Expect(err).To(MatchError(core.ErrRestart))

		Expect(filepath.Join(tempDir, "gamedata/world/levdel.dat")).NotTo(BeAnExistingFile())
	})
})

func Test(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "WorldMiddleware Suite")
}
