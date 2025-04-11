package serverjar_test

import (
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/jarcoal/httpmock"
	"github.com/kofuk/premises/internal/mc/launchermeta"
	"github.com/kofuk/premises/runner/internal/commands/mclauncher/core"
	"github.com/kofuk/premises/runner/internal/commands/mclauncher/middleware/serverjar"
	"github.com/kofuk/premises/runner/internal/env"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func mustResponder(responder httpmock.Responder, err error) httpmock.Responder {
	if err != nil {
		panic(err)
	}
	return responder
}

func TestServerJarMiddleware(t *testing.T) {
	t.Parallel()

	httpmock.Activate(t)
	httpmock.RegisterResponder(
		http.MethodGet,
		"http://launchermeta.premises.local/version_manifest.json",
		mustResponder(httpmock.NewJsonResponder(http.StatusOK, map[string]any{
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
		})),
	)
	httpmock.RegisterResponder(
		http.MethodGet,
		"http://launchermeta.premises.local/1.20.1.json",
		mustResponder(httpmock.NewJsonResponder(http.StatusOK, map[string]any{
			"downloads": map[string]any{
				"server": map[string]any{
					"url": "http://launchermeta.premises.local/1.20.1.jar",
				},
			},
		})),
	)
	httpmock.RegisterResponder(
		http.MethodGet,
		"http://launchermeta.premises.local/1.20.1.jar",
		httpmock.NewStringResponder(http.StatusOK, "#!/usr/bin/true\n"),
	)

	tempDir, err := os.MkdirTemp("", "serverjar_test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	if err := os.MkdirAll(filepath.Join(tempDir, "servers.d"), 0755); err != nil {
		t.Fatalf("failed to create servers.d dir: %v", err)
	}

	ctrl := gomock.NewController(t)
	settings := core.NewMockSettingsRepository(ctrl)
	settings.EXPECT().GetMinecraftVersion().AnyTimes().Return("1.20.1")
	settings.EXPECT().SetServerPath(gomock.Eq(filepath.Join(tempDir, "servers.d/1.20.1.jar"))).Times(1)

	envProvider := env.NewMockEnvProvider(ctrl)
	envProvider.EXPECT().GetDataPath(gomock.Eq("servers.d"), gomock.Eq("1.20.1.jar")).AnyTimes().Return(filepath.Join(tempDir, "servers.d/1.20.1.jar"))

	stateRepository := core.NewMockStateRepository(ctrl)
	stateRepository.EXPECT().GetState(gomock.Eq(serverjar.StateKeyMinecraftVersion)).Return("1.20.1")
	stateRepository.EXPECT().SetState(gomock.Eq(serverjar.StateKeyMinecraftVersion), gomock.Eq("1.20.1")).Return(nil)

	launcherMetaClient := launchermeta.NewLauncherMetaClient(launchermeta.WithManifestURL("http://launchermeta.premises.local/version_manifest.json"))

	launcher := core.NewLauncherCore(settings, envProvider, stateRepository)
	launcher.Middleware(core.StopMiddleware)
	launcher.Middleware(serverjar.NewServerJarMiddleware(launcherMetaClient, http.DefaultClient))

	err = launcher.Start(t.Context())
	assert.NoError(t, err, "ServerJarMiddleware should not return an error")

	assert.FileExists(t, filepath.Join(tempDir, "servers.d/1.20.1.jar"), "Server jar file should exist")
	content, err := os.ReadFile(filepath.Join(tempDir, "servers.d/1.20.1.jar"))
	if err != nil {
		t.Fatalf("failed to read server jar file: %v", err)
	}
	assert.Contains(t, string(content), "#!/usr/bin/true", "Server jar file should contain '#!/usr/bin/true'")
}

func TestServerJarMiddleware_shouldNotDownloadServerJarIfExists(t *testing.T) {
	t.Parallel()

	tempDir, err := os.MkdirTemp("", "serverjar_test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	if err := os.MkdirAll(filepath.Join(tempDir, "servers.d"), 0755); err != nil {
		t.Fatalf("failed to create servers.d dir: %v", err)
	}

	if err := os.WriteFile(filepath.Join(tempDir, "servers.d/1.20.1.jar"), []byte("foo"), 0644); err != nil {
		t.Fatalf("failed to create server jar file: %v", err)
	}

	ctrl := gomock.NewController(t)
	settings := core.NewMockSettingsRepository(ctrl)
	settings.EXPECT().GetMinecraftVersion().AnyTimes().Return("1.20.1")
	settings.EXPECT().SetServerPath(gomock.Eq(filepath.Join(tempDir, "servers.d/1.20.1.jar"))).Times(1)

	envProvider := env.NewMockEnvProvider(ctrl)
	envProvider.EXPECT().GetDataPath(gomock.Eq("servers.d"), gomock.Eq("1.20.1.jar")).AnyTimes().Return(filepath.Join(tempDir, "servers.d/1.20.1.jar"))

	stateRepository := core.NewMockStateRepository(ctrl)
	stateRepository.EXPECT().GetState(gomock.Eq(serverjar.StateKeyMinecraftVersion)).Return("1.20.1")
	stateRepository.EXPECT().SetState(gomock.Eq(serverjar.StateKeyMinecraftVersion), gomock.Eq("1.20.1")).Return(nil)

	launcherMetaClient := launchermeta.NewLauncherMetaClient(launchermeta.WithManifestURL("http://launchermeta.premises.local/version_manifest.json"))

	launcher := core.NewLauncherCore(settings, envProvider, stateRepository)
	launcher.Middleware(core.StopMiddleware)
	launcher.Middleware(serverjar.NewServerJarMiddleware(launcherMetaClient, http.DefaultClient))

	err = launcher.Start(t.Context())
	assert.NoError(t, err, "ServerJarMiddleware should not return an error")

	assert.FileExists(t, filepath.Join(tempDir, "servers.d/1.20.1.jar"), "Server jar file should exist")
	content, err := os.ReadFile(filepath.Join(tempDir, "servers.d/1.20.1.jar"))
	if err != nil {
		t.Fatalf("failed to read server jar file: %v", err)
	}
	assert.Contains(t, string(content), "foo", "Server jar file should contain 'foo'")
}

func TestServerJarMiddleware_directoryShouldBeCleanedWhenVersionChanged(t *testing.T) {
	t.Parallel()

	tempDir, err := os.MkdirTemp("", "serverjar_test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	for _, dir := range []string{"servers.d", "gamedata", "gamedata/world", "gamedata/ss@1", "gamedata/foo"} {
		if err := os.MkdirAll(filepath.Join(tempDir, dir), 0755); err != nil {
			t.Fatalf("failed to create %s dir: %v", dir, err)
		}
	}

	if err := os.WriteFile(filepath.Join(tempDir, "servers.d/1.20.1.jar"), []byte("foo"), 0644); err != nil {
		t.Fatalf("failed to create server jar file: %v", err)
	}
	for _, file := range []string{"server.properties", "world/level.dat", "ss@1/level.dat", "bar.txt"} {
		if err := os.WriteFile(filepath.Join(tempDir, "gamedata", file), []byte("bar"), 0644); err != nil {
			t.Fatalf("failed to create %s file: %v", file, err)
		}
	}

	ctrl := gomock.NewController(t)
	settings := core.NewMockSettingsRepository(ctrl)
	settings.EXPECT().GetMinecraftVersion().AnyTimes().Return("1.20.1")
	settings.EXPECT().SetServerPath(gomock.Eq(filepath.Join(tempDir, "servers.d/1.20.1.jar"))).Times(1)

	envProvider := env.NewMockEnvProvider(ctrl)
	envProvider.EXPECT().GetDataPath(gomock.Eq("servers.d"), gomock.Eq("1.20.1.jar")).AnyTimes().Return(filepath.Join(tempDir, "servers.d/1.20.1.jar"))
	envProvider.EXPECT().GetDataPath(gomock.Eq("gamedata")).AnyTimes().Return(filepath.Join(tempDir, "gamedata"))

	stateRepository := core.NewMockStateRepository(ctrl)
	stateRepository.EXPECT().GetState(gomock.Eq(serverjar.StateKeyMinecraftVersion)).Return("1.20.0")
	stateRepository.EXPECT().SetState(gomock.Eq(serverjar.StateKeyMinecraftVersion), gomock.Eq("1.20.1")).Return(nil)

	launcherMetaClient := launchermeta.NewLauncherMetaClient(launchermeta.WithManifestURL("http://launchermeta.premises.local/version_manifest.json"))

	launcher := core.NewLauncherCore(settings, envProvider, stateRepository)
	launcher.Middleware(core.StopMiddleware)
	launcher.Middleware(serverjar.NewServerJarMiddleware(launcherMetaClient, http.DefaultClient))

	err = launcher.Start(t.Context())
	assert.NoError(t, err, "ServerJarMiddleware should not return an error")

	assert.FileExists(t, filepath.Join(tempDir, "servers.d/1.20.1.jar"), "Server jar file should exist")
	content, err := os.ReadFile(filepath.Join(tempDir, "servers.d/1.20.1.jar"))
	if err != nil {
		t.Fatalf("failed to read server jar file: %v", err)
	}
	assert.Contains(t, string(content), "foo", "Server jar file should contain 'foo'")

	assert.NoDirExists(t, filepath.Join(tempDir, "gamedata/foo"), "foo should be deleted")
	assert.NoFileExists(t, filepath.Join(tempDir, "gamedata/bar.txt"), "unneeded file should be deleted")
	assert.FileExists(t, filepath.Join(tempDir, "gamedata/server.properties"), "server.properties should not be deleted")
	assert.FileExists(t, filepath.Join(tempDir, "gamedata/world/level.dat"), "world/level.dat should not be deleted")
	assert.FileExists(t, filepath.Join(tempDir, "gamedata/ss@1/level.dat"), "ss@1/level.dat should not be deleted")
}

func TestServerJarMiddleware_directoryShouldNotBeCleanedWhenVersionNotChanged(t *testing.T) {
	t.Parallel()

	tempDir, err := os.MkdirTemp("", "serverjar_test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	for _, dir := range []string{"servers.d", "gamedata", "gamedata/world", "gamedata/ss@1", "gamedata/foo"} {
		if err := os.MkdirAll(filepath.Join(tempDir, dir), 0755); err != nil {
			t.Fatalf("failed to create %s dir: %v", dir, err)
		}
	}

	if err := os.WriteFile(filepath.Join(tempDir, "servers.d/1.20.1.jar"), []byte("foo"), 0644); err != nil {
		t.Fatalf("failed to create server jar file: %v", err)
	}
	for _, file := range []string{"server.properties", "world/level.dat", "ss@1/level.dat", "bar.txt"} {
		if err := os.WriteFile(filepath.Join(tempDir, "gamedata", file), []byte("bar"), 0644); err != nil {
			t.Fatalf("failed to create %s file: %v", file, err)
		}
	}

	ctrl := gomock.NewController(t)
	settings := core.NewMockSettingsRepository(ctrl)
	settings.EXPECT().GetMinecraftVersion().AnyTimes().Return("1.20.1")
	settings.EXPECT().SetServerPath(gomock.Eq(filepath.Join(tempDir, "servers.d/1.20.1.jar"))).Times(1)

	envProvider := env.NewMockEnvProvider(ctrl)
	envProvider.EXPECT().GetDataPath(gomock.Eq("servers.d"), gomock.Eq("1.20.1.jar")).AnyTimes().Return(filepath.Join(tempDir, "servers.d/1.20.1.jar"))
	envProvider.EXPECT().GetDataPath(gomock.Eq("gamedata")).AnyTimes().Return(filepath.Join(tempDir, "gamedata"))

	stateRepository := core.NewMockStateRepository(ctrl)
	stateRepository.EXPECT().GetState(gomock.Eq(serverjar.StateKeyMinecraftVersion)).Return("1.20.1")
	stateRepository.EXPECT().SetState(gomock.Eq(serverjar.StateKeyMinecraftVersion), gomock.Eq("1.20.1")).Return(nil)

	launcherMetaClient := launchermeta.NewLauncherMetaClient(launchermeta.WithManifestURL("http://launchermeta.premises.local/version_manifest.json"))

	launcher := core.NewLauncherCore(settings, envProvider, stateRepository)
	launcher.Middleware(core.StopMiddleware)
	launcher.Middleware(serverjar.NewServerJarMiddleware(launcherMetaClient, http.DefaultClient))

	err = launcher.Start(t.Context())
	assert.NoError(t, err, "ServerJarMiddleware should not return an error")

	assert.FileExists(t, filepath.Join(tempDir, "servers.d/1.20.1.jar"), "Server jar file should exist")
	content, err := os.ReadFile(filepath.Join(tempDir, "servers.d/1.20.1.jar"))
	if err != nil {
		t.Fatalf("failed to read server jar file: %v", err)
	}
	assert.Contains(t, string(content), "foo", "Server jar file should contain 'foo'")

	assert.DirExists(t, filepath.Join(tempDir, "gamedata/foo"), "foo should be present")
	assert.FileExists(t, filepath.Join(tempDir, "gamedata/bar.txt"), "unneeded file should not be deleted")
	assert.FileExists(t, filepath.Join(tempDir, "gamedata/server.properties"), "server.properties should not be deleted")
	assert.FileExists(t, filepath.Join(tempDir, "gamedata/world/level.dat"), "world/level.dat should not be deleted")
	assert.FileExists(t, filepath.Join(tempDir, "gamedata/ss@1/level.dat"), "ss@1/level.dat should not be deleted")
}
