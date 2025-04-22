package serverjar_test

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/jarcoal/httpmock"
	"github.com/kofuk/premises/internal/mc/launchermeta"
	"github.com/kofuk/premises/runner/internal/commands/mclauncher/core"
	"github.com/kofuk/premises/runner/internal/commands/mclauncher/middleware/serverjar"
	"github.com/kofuk/premises/runner/internal/env"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"
)

var _ = Describe("ServerJarMiddleware", func() {
	var (
		ctrl               *gomock.Controller
		settingsRepository *core.MockSettingsRepository
		envProvider        *env.MockEnvProvider
		stateRepository    *core.MockStateRepository
		launcherMetaClient *launchermeta.LauncherMetaClient
		launcher           *core.LauncherCore
		tempDir            string
	)

	BeforeEach(func() {
		httpmock.Activate(GinkgoTB())
		ctrl = gomock.NewController(GinkgoT())

		httpmock.RegisterResponder(
			http.MethodGet,
			"http://launchermeta.premises.local/version_manifest.json",
			httpmock.NewJsonResponderOrPanic(http.StatusOK, map[string]any{
				"latest": map[string]any{
					"release":  "1.20.1",
					"snapshot": "1.20.1",
				},
				"versions": []any{
					map[string]any{
						"id":          "1.20.1",
						"type":        "release",
						"url":         "http://launchermeta.premises.local/1.20.1.json",
						"releaseTime": "2022-12-30T10:35:10+00:00",
					},
				},
			}),
		)
		httpmock.RegisterResponder(
			http.MethodGet,
			"http://launchermeta.premises.local/1.20.1.json",
			httpmock.NewJsonResponderOrPanic(http.StatusOK, map[string]any{
				"downloads": map[string]any{
					"server": map[string]any{
						"url": "http://launchermeta.premises.local/1.20.1.jar",
					},
				},
			}),
		)
		httpmock.RegisterResponder(
			http.MethodGet,
			"http://launchermeta.premises.local/1.20.1.jar",
			httpmock.NewStringResponder(http.StatusOK, "#!/usr/bin/true\n"),
		)

		settingsRepository = core.NewMockSettingsRepository(ctrl)
		envProvider = env.NewMockEnvProvider(ctrl)
		stateRepository = core.NewMockStateRepository(ctrl)
		launcherMetaClient = launchermeta.NewLauncherMetaClient(launchermeta.WithManifestURL("http://launchermeta.premises.local/version_manifest.json"))

		launcher = core.NewLauncherCore(settingsRepository, envProvider, stateRepository)
		launcher.Use(core.StopMiddleware)

		tempDir = GinkgoT().TempDir()

		if err := os.MkdirAll(filepath.Join(tempDir, "servers.d"), 0o755); err != nil {
			Fail(fmt.Sprintf("failed to create servers.d dir: %v", err))
		}
	})

	It("should download server.jar", func() {
		settingsRepository.EXPECT().GetMinecraftVersion().AnyTimes().Return("1.20.1")
		settingsRepository.EXPECT().SetServerPath(gomock.Eq(filepath.Join(tempDir, "servers.d/1.20.1.jar"))).Times(1)

		envProvider.EXPECT().GetDataPath(gomock.Eq("servers.d"), gomock.Eq("1.20.1.jar")).AnyTimes().Return(filepath.Join(tempDir, "servers.d/1.20.1.jar"))

		stateRepository.EXPECT().GetState(gomock.Any(), gomock.Eq(serverjar.StateKeyMinecraftVersion)).Return("1.20.1", nil)
		stateRepository.EXPECT().SetState(gomock.Any(), gomock.Eq(serverjar.StateKeyMinecraftVersion), gomock.Eq("1.20.1")).Return(nil)

		sut := serverjar.NewServerJarMiddleware(launcherMetaClient, http.DefaultClient)

		launcher.Use(sut)

		err := launcher.Start(GinkgoT().Context())
		Expect(err).ShouldNot(HaveOccurred(), "ServerJarMiddleware should not return an error")

		Expect(filepath.Join(tempDir, "servers.d/1.20.1.jar")).To(BeARegularFile(), "Server jar file should exist")
		content, _ := os.ReadFile(filepath.Join(tempDir, "servers.d/1.20.1.jar"))
		Expect(string(content)).To(Equal("#!/usr/bin/true\n"), "Server jar file should contain '#!/usr/bin/true'")
	})

	It("should not download server.jar if desired version already exists", func() {
		os.WriteFile(filepath.Join(tempDir, "servers.d/1.20.1.jar"), []byte("foo"), 0o644)

		settingsRepository.EXPECT().GetMinecraftVersion().AnyTimes().Return("1.20.1")
		settingsRepository.EXPECT().SetServerPath(gomock.Eq(filepath.Join(tempDir, "servers.d/1.20.1.jar"))).Times(1)

		envProvider.EXPECT().GetDataPath(gomock.Eq("servers.d"), gomock.Eq("1.20.1.jar")).AnyTimes().Return(filepath.Join(tempDir, "servers.d/1.20.1.jar"))

		stateRepository.EXPECT().GetState(gomock.Any(), gomock.Eq(serverjar.StateKeyMinecraftVersion)).Return("1.20.1", nil)
		stateRepository.EXPECT().SetState(gomock.Any(), gomock.Eq(serverjar.StateKeyMinecraftVersion), gomock.Eq("1.20.1")).Return(nil)

		sut := serverjar.NewServerJarMiddleware(launcherMetaClient, http.DefaultClient)

		launcher.Use(sut)

		err := launcher.Start(GinkgoT().Context())
		Expect(err).ShouldNot(HaveOccurred(), "ServerJarMiddleware should not return an error")

		Expect(filepath.Join(tempDir, "servers.d/1.20.1.jar")).To(BeARegularFile(), "Server jar file should exist")
		content, _ := os.ReadFile(filepath.Join(tempDir, "servers.d/1.20.1.jar"))
		Expect(string(content)).To(Equal("foo"), "Server jar file should contain 'foo'")
	})

	It("should clean up data directory when version changes", func() {
		for _, dir := range []string{"gamedata", "gamedata/world", "gamedata/ss@1", "gamedata/foo"} {
			os.MkdirAll(filepath.Join(tempDir, dir), 0o755)
		}

		os.WriteFile(filepath.Join(tempDir, "servers.d/1.20.1.jar"), []byte("foo"), 0o644)
		for _, file := range []string{"server.properties", "world/level.dat", "ss@1/level.dat", "bar.txt"} {
			os.WriteFile(filepath.Join(tempDir, "gamedata", file), []byte("bar"), 0644)
		}

		settingsRepository.EXPECT().GetMinecraftVersion().AnyTimes().Return("1.20.1")
		settingsRepository.EXPECT().SetServerPath(gomock.Eq(filepath.Join(tempDir, "servers.d/1.20.1.jar"))).Times(1)

		envProvider.EXPECT().GetDataPath(gomock.Eq("servers.d"), gomock.Eq("1.20.1.jar")).AnyTimes().Return(filepath.Join(tempDir, "servers.d/1.20.1.jar"))
		envProvider.EXPECT().GetDataPath(gomock.Eq("gamedata")).AnyTimes().Return(filepath.Join(tempDir, "gamedata"))

		stateRepository.EXPECT().GetState(gomock.Any(), gomock.Eq(serverjar.StateKeyMinecraftVersion)).Return("1.20.0", nil)
		stateRepository.EXPECT().SetState(gomock.Any(), gomock.Eq(serverjar.StateKeyMinecraftVersion), gomock.Eq("1.20.1")).Return(nil)

		sut := serverjar.NewServerJarMiddleware(launcherMetaClient, http.DefaultClient)

		launcher.Use(sut)

		err := launcher.Start(GinkgoT().Context())
		Expect(err).ShouldNot(HaveOccurred(), "ServerJarMiddleware should not return an error")

		Expect(filepath.Join(tempDir, "servers.d/1.20.1.jar")).To(BeARegularFile(), "Server jar file should exist")
		content, _ := os.ReadFile(filepath.Join(tempDir, "servers.d/1.20.1.jar"))
		Expect(string(content)).To(Equal("foo"), "Server jar file should contain 'foo'")

		Expect(filepath.Join(tempDir, "gamedata/foo")).NotTo(BeAnExistingFile(), "foo should be deleted")
		Expect(filepath.Join(tempDir, "gamedata/bar.txt")).NotTo(BeAnExistingFile(), "unneeded file should be deleted")
		Expect(filepath.Join(tempDir, "gamedata/server.properties")).To(BeAnExistingFile(), "server.properties should not be deleted")
		Expect(filepath.Join(tempDir, "gamedata/world/level.dat")).To(BeAnExistingFile(), "world data should not be deleted")
		Expect(filepath.Join(tempDir, "gamedata/ss@1/level.dat")).To(BeAnExistingFile(), "snapshots should not be deleted")
	})

	It("should not clean up data directory when version does not change", func() {
		for _, dir := range []string{"gamedata", "gamedata/world", "gamedata/ss@1", "gamedata/foo"} {
			os.MkdirAll(filepath.Join(tempDir, dir), 0755)
		}

		os.WriteFile(filepath.Join(tempDir, "servers.d/1.20.1.jar"), []byte("foo"), 0o644)
		for _, file := range []string{"server.properties", "world/level.dat", "ss@1/level.dat", "bar.txt"} {
			os.WriteFile(filepath.Join(tempDir, "gamedata", file), []byte("bar"), 0644)
		}

		settingsRepository.EXPECT().GetMinecraftVersion().AnyTimes().Return("1.20.1")
		settingsRepository.EXPECT().SetServerPath(gomock.Eq(filepath.Join(tempDir, "servers.d/1.20.1.jar"))).Times(1)

		envProvider.EXPECT().GetDataPath(gomock.Eq("servers.d"), gomock.Eq("1.20.1.jar")).AnyTimes().Return(filepath.Join(tempDir, "servers.d/1.20.1.jar"))
		envProvider.EXPECT().GetDataPath(gomock.Eq("gamedata")).AnyTimes().Return(filepath.Join(tempDir, "gamedata"))

		stateRepository.EXPECT().GetState(gomock.Any(), gomock.Eq(serverjar.StateKeyMinecraftVersion)).Return("1.20.1", nil)
		stateRepository.EXPECT().SetState(gomock.Any(), gomock.Eq(serverjar.StateKeyMinecraftVersion), gomock.Eq("1.20.1")).Return(nil)

		sut := serverjar.NewServerJarMiddleware(launcherMetaClient, http.DefaultClient)

		launcher.Use(sut)

		err := launcher.Start(GinkgoT().Context())
		Expect(err).ShouldNot(HaveOccurred(), "ServerJarMiddleware should not return an error")

		Expect(filepath.Join(tempDir, "servers.d/1.20.1.jar")).To(BeARegularFile(), "Server jar file should exist")
		content, _ := os.ReadFile(filepath.Join(tempDir, "servers.d/1.20.1.jar"))
		Expect(string(content)).To(Equal("foo"), "Server jar file should contain 'foo'")

		Expect(filepath.Join(tempDir, "gamedata/foo")).To(BeADirectory(), "foo should be present")
		Expect(filepath.Join(tempDir, "gamedata/bar.txt")).To(BeARegularFile(), "unneeded file should not be deleted")
		Expect(filepath.Join(tempDir, "gamedata/server.properties")).To(BeARegularFile(), "server.properties should not be deleted")
		Expect(filepath.Join(tempDir, "gamedata/world/level.dat")).To(BeARegularFile(), "world data should not be deleted")
		Expect(filepath.Join(tempDir, "gamedata/ss@1/level.dat")).To(BeARegularFile(), "snapshots should not be deleted")
	})
})

func Test(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "ServerJarMiddleware Suite")
}
