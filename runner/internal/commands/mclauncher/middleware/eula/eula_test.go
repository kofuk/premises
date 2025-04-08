package eula_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/kofuk/premises/runner/internal/commands/mclauncher/core"
	"github.com/kofuk/premises/runner/internal/commands/mclauncher/middleware/eula"
	"github.com/kofuk/premises/runner/internal/env"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestEulaMiddleware(t *testing.T) {
	t.Parallel()

	tempDir, err := os.MkdirTemp("", "eula_test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)
	ctrl := gomock.NewController(t)
	settings := core.NewMockSettingsRepository(ctrl)

	envProvider := env.NewMockEnvProvider(ctrl)
	envProvider.EXPECT().GetDataPath(gomock.Any()).AnyTimes().Return(tempDir)

	launcher := core.New(settings, envProvider)
	launcher.Middleware(core.StopMiddleware)
	launcher.Middleware(eula.NewEulaMiddleware())

	err = launcher.Start(t.Context())
	assert.NoError(t, err, "EulaMiddleware should not return an error")

	assert.FileExists(t, filepath.Join(tempDir, "eula.txt"), "EULA file should exist")
	content, err := os.ReadFile(filepath.Join(tempDir, "eula.txt"))
	if err != nil {
		t.Fatalf("failed to read eula.txt: %v", err)
	}
	assert.Contains(t, string(content), "eula=true", "EULA file should contain 'eula=true'")
}
