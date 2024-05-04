package mclauncher

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/kofuk/premises/common/entity"
	"github.com/kofuk/premises/common/entity/runner"
	"github.com/kofuk/premises/runner/commands/mclauncher/gamesrv"
	"github.com/kofuk/premises/runner/exterior"
	"github.com/kofuk/premises/runner/fs"
	"github.com/kofuk/premises/runner/rpc"
	"github.com/kofuk/premises/runner/rpc/types"
)

type RPCHandler struct {
	s    *rpc.Server
	game *gamesrv.ServerInstance
}

func NewRPCHandler(s *rpc.Server, game *gamesrv.ServerInstance) *RPCHandler {
	return &RPCHandler{
		s:    s,
		game: game,
	}
}

func (h *RPCHandler) HandleGameStop(req *rpc.AbstractRequest) (any, error) {
	h.game.ShouldStop = true
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

	if err := os.WriteFile(fs.LocateDataFile("config.json"), data, 0644); err != nil {
		slog.Error("Failed to write server config", slog.Any("error", err))
		return nil, err
	}

	h.game.RestartRequested = true
	h.game.Stop()

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

		if err := exterior.DispatchMessage("serverStatus", runner.Event{
			Type: runner.EventInfo,
			Info: &runner.InfoExtra{
				InfoCode: entity.InfoSnapshotError,
				Actor:    input.Actor,
				IsError:  true,
			},
		}); err != nil {
			slog.Error("Unable to write send message", slog.Any("error", err))
		}

		return nil, err
	}

	if err := exterior.DispatchMessage("serverStatus", runner.Event{
		Type: runner.EventInfo,
		Info: &runner.InfoExtra{
			InfoCode: entity.InfoSnapshotDone,
			Actor:    input.Actor,
			IsError:  false,
		},
	}); err != nil {
		slog.Error("Unable to write send message", slog.Any("error", err))
	}

	return "ok", nil
}

func (h *RPCHandler) HandleSnapshotUndo(req *rpc.AbstractRequest) (any, error) {
	var input types.SnapshotInput
	if err := req.Bind(&input); err != nil {
		return nil, err
	}

	go func() {
		if _, err := os.Stat(filepath.Join(fs.LocateWorldData(fmt.Sprintf("ss@quick%d/world", input.Slot)))); err != nil {
			if err := exterior.DispatchMessage("serverStatus", runner.Event{
				Type: runner.EventInfo,
				Info: &runner.InfoExtra{
					InfoCode: entity.InfoNoSnapshot,
					Actor:    input.Actor,
					IsError:  true,
				},
			}); err != nil {
				slog.Error("Unable to write send message", slog.Any("error", err))
			}
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
