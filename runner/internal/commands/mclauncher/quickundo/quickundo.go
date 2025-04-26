package quickundo

import (
	"context"
	"os"
	"path/filepath"

	"github.com/kofuk/premises/runner/internal/commands/mclauncher/core"
	"github.com/kofuk/premises/runner/internal/fs"
	"github.com/kofuk/premises/runner/internal/rpc"
	"github.com/kofuk/premises/runner/internal/rpc/types"
)

type QuickUndoService struct {
	rpcClient   *rpc.Client
	executor    *KillableCommandExecutor
	restorePath string
}

func NewQuickUndoService(rpcClient *rpc.Client) *QuickUndoService {
	return &QuickUndoService{
		rpcClient: rpcClient,
		executor:  NewKillableCommandExecutor(),
	}
}

func (s *QuickUndoService) BeforeLaunch(c *core.LauncherContext) error {
	if s.restorePath == "" {
		// We are not in a state to restore a snapshot.
		return nil
	}

	restorePath := s.restorePath
	s.restorePath = ""

	worldDir := c.Env().GetDataPath("gamedata/world")
	if err := os.RemoveAll(worldDir); err != nil {
		return err
	}

	if err := os.Mkdir(worldDir, 0755); err != nil {
		return err
	}

	if err := fs.CopyAll(filepath.Join(restorePath, "world"), worldDir); err != nil {
		return err
	}

	return nil
}

func (s *QuickUndoService) Register(launcher *core.LauncherCore) {
	launcher.CommandExecutor = s.executor
	launcher.AddBeforeLaunchListener(s.BeforeLaunch)
}

func (s *QuickUndoService) CreateSnapshot(ctx context.Context, slot int) error {
	return s.rpcClient.Call(ctx, "snapshot/create", types.SnapshotHelperInput{
		Slot: slot,
	}, nil)
}

func (s *QuickUndoService) RestartWithSnapshot(ctx context.Context, slot int) error {
	var snapshotInfo types.SnapshotHelperOutput
	err := s.rpcClient.Call(ctx, "snapshot/stat", types.SnapshotHelperInput{
		Slot: slot,
	}, &snapshotInfo)
	if err != nil {
		return err
	}

	s.restorePath = snapshotInfo.Path

	return s.executor.Kill()
}
