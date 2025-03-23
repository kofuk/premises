package core_test

import (
	"errors"
	"testing"

	"github.com/kofuk/premises/runner/internal/commands/mclauncher/core"
	"github.com/kofuk/premises/runner/internal/system"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestLaunch(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	executor := system.NewMockCommandExecutor(ctrl)
	settings := core.NewMockSettingsRepository(ctrl)
	settings.EXPECT().GetServerPath().Return("/usr/bin/true")
	settings.EXPECT().GetDataDir().Return("/tmp")
	executor.EXPECT().Run(gomock.Any(), "/usr/bin/true", []string{}, gomock.Any()).Times(1).Return(nil)

	launcher := core.New(settings)
	launcher.CommandExecutor = executor

	err := launcher.Start(t.Context())

	assert.NoError(t, err)
}

func TestServerFailure(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	executor := system.NewMockCommandExecutor(ctrl)
	settings := core.NewMockSettingsRepository(ctrl)
	settings.EXPECT().GetServerPath().Return("/usr/bin/false")
	settings.EXPECT().GetDataDir().Return("/tmp")
	gomock.InOrder(
		executor.EXPECT().Run(gomock.Any(), "/usr/bin/false", []string{}, gomock.Any()).Times(2).Return(errors.New("error")),
		executor.EXPECT().Run(gomock.Any(), "/usr/bin/false", []string{}, gomock.Any()).Return(nil),
	)

	launcher := core.New(settings)
	launcher.CommandExecutor = executor

	err := launcher.Start(t.Context())

	assert.NoError(t, err)
}
