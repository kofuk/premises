package privileged

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/kofuk/premises/runner/fs"
	"github.com/kofuk/premises/runner/rpc"
	"github.com/kofuk/premises/runner/rpc/types"
	"github.com/kofuk/premises/runner/systemutil"
)

type SnapshotInfo struct {
	ID   string `json:"id"`
	Path string `json:"path"`
}

func takeFsSnapshot(snapshotId string) (*SnapshotInfo, error) {
	var snapshotInfo SnapshotInfo
	snapshotInfo.ID = snapshotId
	snapshotInfo.Path = fs.DataPath("gamedata/ss@" + snapshotId)

	if _, err := os.Stat(snapshotInfo.Path); err == nil {
		if err := deleteFsSnapshot(snapshotId); err != nil {
			slog.Error("Failed to remove old snapshot (doesn't the snapshot exist?)", slog.Any("error", err))
		}
	}

	// Create read-only snapshot
	if err := systemutil.Cmd("btrfs", []string{"subvolume", "snapshot", "-r", ".", snapshotInfo.Path}, systemutil.WithWorkingDir(fs.DataPath("gamedata"))); err != nil {
		return nil, err
	}

	return &snapshotInfo, nil
}

func deleteFsSnapshot(id string) error {
	if strings.Contains(id, "/") {
		return errors.New("invalid snapshot ID")
	}

	if err := systemutil.Cmd("btrfs", []string{"subvolume", "delete", "ss@" + id}, systemutil.WithWorkingDir(fs.DataPath("gamedata"))); err != nil {
		return err
	}
	if err := systemutil.Cmd("btrfs", []string{"balance", "."}, systemutil.WithWorkingDir(fs.DataPath("gamedata"))); err != nil {
		return err
	}

	return nil
}

func Run(args []string) int {
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

	<-make(chan struct{})

	return 0
}
