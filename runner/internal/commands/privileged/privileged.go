package privileged

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/kofuk/premises/runner/internal/env"
	"github.com/kofuk/premises/runner/internal/rpc"
	"github.com/kofuk/premises/runner/internal/rpc/types"
	"github.com/kofuk/premises/runner/internal/system"
)

type SnapshotInfo struct {
	ID   string `json:"id"`
	Path string `json:"path"`
}

func takeFsSnapshot(snapshotId string) (*SnapshotInfo, error) {
	var snapshotInfo SnapshotInfo
	snapshotInfo.ID = snapshotId
	snapshotInfo.Path = env.DataPath("gamedata/ss@" + snapshotId)

	if _, err := os.Stat(snapshotInfo.Path); err == nil {
		if err := deleteFsSnapshot(snapshotId); err != nil {
			slog.Error("Failed to remove old snapshot (doesn't the snapshot exist?)", slog.Any("error", err))
		}
	}

	// Create read-only snapshot
	if err := system.Cmd("btrfs", []string{"subvolume", "snapshot", "-r", ".", snapshotInfo.Path}, system.WithWorkingDir(env.DataPath("gamedata"))); err != nil {
		return nil, err
	}

	return &snapshotInfo, nil
}

func deleteFsSnapshot(id string) error {
	if strings.Contains(id, "/") {
		return errors.New("invalid snapshot ID")
	}

	if err := system.Cmd("btrfs", []string{"subvolume", "delete", "ss@" + id}, system.WithWorkingDir(env.DataPath("gamedata"))); err != nil {
		return err
	}
	if err := system.Cmd("btrfs", []string{"balance", "."}, system.WithWorkingDir(env.DataPath("gamedata"))); err != nil {
		return err
	}

	return nil
}

func Run(args []string) int {
	rpc.ToExteriord.Notify("proc/registerStopHook", os.Getenv("PREMISES_RUNNER_COMMAND"))

	ctx, cancelFn := context.WithCancel(context.Background())

	rpc.DefaultServer.RegisterMethod("snapshot/create", func(req *rpc.AbstractRequest) (any, error) {
		var ss types.SnapshotHelperInput
		if err := req.Bind(&ss); err != nil {
			return nil, err
		}

		info, err := takeFsSnapshot(fmt.Sprintf("quick%d", ss.Slot))
		if err != nil {
			return nil, err
		}

		return types.SnapshotHelperOutput{
			ID:   info.ID,
			Path: info.Path,
		}, nil
	})
	rpc.DefaultServer.RegisterNotifyMethod("base/stop", func(req *rpc.AbstractRequest) error {
		cancelFn()
		return nil
	})

	<-ctx.Done()

	return 0
}
