package mclauncher

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/kofuk/premises/internal/entity"
	"github.com/kofuk/premises/internal/entity/runner"
	"github.com/kofuk/premises/runner/internal/commands/mclauncher/game"
	"github.com/kofuk/premises/runner/internal/env"
	"github.com/kofuk/premises/runner/internal/exterior"
	"github.com/kofuk/premises/runner/internal/rpc"
	"github.com/kofuk/premises/runner/internal/rpc/types"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
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

func (h *RPCHandler) HandleGameStop(ctx context.Context, req *rpc.AbstractRequest) error {
	h.game.Stop(ctx)
	return nil
}

func (h *RPCHandler) HandleSnapshotCreate(ctx context.Context, req *rpc.AbstractRequest) error {
	var input types.SnapshotInput
	if err := req.Bind(&input); err != nil {
		return err
	}

	go func() {
		if err := h.game.SaveAll(); err != nil {
			slog.Error("Failed to run save-all", slog.Any("error", err))
			return
		}

		if err := rpc.ToSnapshotHelper.Call(ctx, "snapshot/create", types.SnapshotHelperInput{
			Slot: input.Slot,
		}, nil); err != nil {
			slog.Error("Failed to create snapshot", slog.Any("error", err))

			exterior.DispatchEvent(ctx, runner.Event{
				Type: runner.EventInfo,
				Info: &runner.InfoExtra{
					InfoCode: entity.InfoSnapshotError,
					Actor:    input.Actor,
					IsError:  true,
				},
			})

			return
		}

		exterior.DispatchEvent(ctx, runner.Event{
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

func (h *RPCHandler) HandleSnapshotUndo(ctx context.Context, req *rpc.AbstractRequest) error {
	var input types.SnapshotInput
	if err := req.Bind(&input); err != nil {
		return err
	}

	go func() {
		tracer := otel.GetTracerProvider().Tracer(ScopeName)
		ctx := trace.ContextWithSpanContext(context.Background(), trace.SpanContextFromContext(ctx))
		ctx, span := tracer.Start(ctx, "Revert to snapshot")
		defer span.End()

		if _, err := os.Stat(env.DataPath(fmt.Sprintf("gamedata/ss@quick%d/world", input.Slot))); err != nil {
			span.SetStatus(codes.Error, err.Error())

			exterior.DispatchEvent(ctx, runner.Event{
				Type: runner.EventInfo,
				Info: &runner.InfoExtra{
					InfoCode: entity.InfoNoSnapshot,
					Actor:    input.Actor,
					IsError:  true,
				},
			})
			return
		}

		if err := h.game.QuickUndo(ctx, input.Slot); err != nil {
			span.SetStatus(codes.Error, err.Error())

			slog.Error("Unable to quick-undo", slog.Any("error", err))
		}
	}()

	return nil
}

func (h *RPCHandler) Bind() {
	h.s.RegisterNotifyMethod("game/stop", h.HandleGameStop)
	h.s.RegisterNotifyMethod("snapshot/create", h.HandleSnapshotCreate)
	h.s.RegisterNotifyMethod("snapshot/undo", h.HandleSnapshotUndo)
}
