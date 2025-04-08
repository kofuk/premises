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

	launcherMetaClient := launchermeta.NewLauncherMetaClient(launchermeta.WithManifestURL("http://launchermeta.premises.local/version_manifest.json"))

	launcher := core.New(settings, envProvider)
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
