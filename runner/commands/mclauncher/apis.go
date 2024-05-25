package mclauncher

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"

	"github.com/kofuk/premises/internal/entity"
	"github.com/kofuk/premises/internal/entity/runner"
	"github.com/kofuk/premises/runner/commands/mclauncher/game"
	"github.com/kofuk/premises/runner/exterior"
	"github.com/kofuk/premises/runner/fs"
	"github.com/kofuk/premises/runner/rpc"
	"github.com/kofuk/premises/runner/rpc/types"
)

type RPCHandler struct {
	s    *rpc.Server
	game *game.Launcher
}

func NewRPCHandler(s *rpc.Server, game *game.Launcher) *RPCHandler {
	return &RPCHandler{
		s:    s,
		game: game,
	}
}

func (h *RPCHandler) HandleGameStop(req *rpc.AbstractRequest) error {
	h.game.Stop()
	return nil
}

func (h *RPCHandler) HandleGameReconfigure(req *rpc.AbstractRequest) error {
	var config runner.Config
	if err := req.Bind(&config); err != nil {
		return err
	}

	data, err := json.Marshal(&config)
	if err != nil {
		slog.Error("Failed to stringify request", slog.Any("error", err))
		return err
	}

	if err := os.WriteFile(fs.DataPath("config.json"), data, 0644); err != nil {
		slog.Error("Failed to write server config", slog.Any("error", err))
		return err
	}

	h.game.StopToRestart()

	return nil
}

func (h *RPCHandler) HandleSnapshotCreate(req *rpc.AbstractRequest) error {
	var input types.SnapshotInput
	if err := req.Bind(&input); err != nil {
		return err
	}

	go func() {
		if err := h.game.SaveAll(); err != nil {
			slog.Error("Failed to run save-all", slog.Any("error", err))
			return
		}

		if err := rpc.ToSnapshotHelper.Call("snapshot/create", types.SnapshotHelperInput{
			Slot: input.Slot,
		}, nil); err != nil {
			slog.Error("Failed to create snapshot", slog.Any("error", err))

			exterior.DispatchEvent(runner.Event{
				Type: runner.EventInfo,
				Info: &runner.InfoExtra{
					InfoCode: entity.InfoSnapshotError,
					Actor:    input.Actor,
					IsError:  true,
				},
			})

			return
		}

		exterior.DispatchEvent(runner.Event{
			Type: runner.EventInfo,
			Info: &runner.InfoExtra{
				InfoCode: entity.InfoSnapshotDone,
				Actor:    input.Actor,
				IsError:  false,
			},
		})
	}()

	return nil
}

func (h *RPCHandler) HandleSnapshotUndo(req *rpc.AbstractRequest) error {
	var input types.SnapshotInput
	if err := req.Bind(&input); err != nil {
		return err
	}

	go func() {
		if _, err := os.Stat(fs.DataPath(fmt.Sprintf("gamedata/ss@quick%d/world", input.Slot))); err != nil {
			exterior.DispatchEvent(runner.Event{
				Type: runner.EventInfo,
				Info: &runner.InfoExtra{
					InfoCode: entity.InfoNoSnapshot,
					Actor:    input.Actor,
					IsError:  true,
				},
			})
			return
		}

		if err := h.game.QuickUndo(input.Slot); err != nil {
			slog.Error("Unable to quick-undo", slog.Any("error", err))
		}
	}()

	return nil
}

func (h *RPCHandler) Bind() {
	h.s.RegisterNotifyMethod("game/stop", h.HandleGameStop)
	h.s.RegisterNotifyMethod("game/reconfigure", h.HandleGameReconfigure)
	h.s.RegisterNotifyMethod("snapshot/create", h.HandleSnapshotCreate)
	h.s.RegisterNotifyMethod("snapshot/undo", h.HandleSnapshotUndo)
}
