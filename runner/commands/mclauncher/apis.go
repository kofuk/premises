package mclauncher

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"

	"github.com/kofuk/premises/common/entity"
	"github.com/kofuk/premises/common/entity/runner"
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

func (h *RPCHandler) HandleGameStop(req *rpc.AbstractRequest) (any, error) {
	h.game.Stop()
	return "ok", nil
}

func (h *RPCHandler) HandleGameReconfigure(req *rpc.AbstractRequest) (any, error) {
	var config runner.Config
	if err := req.Bind(&config); err != nil {
		return nil, err
	}

	data, err := json.Marshal(&config)
	if err != nil {
		slog.Error("Failed to stringify request", slog.Any("error", err))
		return nil, err
	}

	if err := os.WriteFile(fs.DataPath("config.json"), data, 0644); err != nil {
		slog.Error("Failed to write server config", slog.Any("error", err))
		return nil, err
	}

	h.game.StopToRestart()

	return "ok", nil
}

func (h *RPCHandler) HandleSnapshotCreate(req *rpc.AbstractRequest) (any, error) {
	var input types.SnapshotInput
	if err := req.Bind(&input); err != nil {
		return nil, err
	}

	if err := h.game.SaveAll(); err != nil {
		slog.Error("Failed to run save-all", slog.Any("error", err))
		return nil, err
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

		return nil, err
	}

	exterior.DispatchEvent(runner.Event{
		Type: runner.EventInfo,
		Info: &runner.InfoExtra{
			InfoCode: entity.InfoSnapshotDone,
			Actor:    input.Actor,
			IsError:  false,
		},
	})

	return "ok", nil
}

func (h *RPCHandler) HandleSnapshotUndo(req *rpc.AbstractRequest) (any, error) {
	var input types.SnapshotInput
	if err := req.Bind(&input); err != nil {
		return nil, err
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

	return "accepted", nil
}

func (h *RPCHandler) Bind() {
	h.s.RegisterMethod("game/stop", h.HandleGameStop)
	h.s.RegisterMethod("game/reconfigure", h.HandleGameReconfigure)
	h.s.RegisterMethod("snapshot/create", h.HandleSnapshotCreate)
	h.s.RegisterMethod("snapshot/undo", h.HandleSnapshotUndo)
}
