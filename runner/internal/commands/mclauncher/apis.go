package mclauncher

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/kofuk/premises/internal/entity"
	"github.com/kofuk/premises/internal/entity/runner"
	"github.com/kofuk/premises/runner/internal/commands/mclauncher/quickundo"
	"github.com/kofuk/premises/runner/internal/commands/mclauncher/rcon"
	"github.com/kofuk/premises/runner/internal/exterior"
	"github.com/kofuk/premises/runner/internal/rpc"
	"github.com/kofuk/premises/runner/internal/rpc/types"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

type RPCHandler struct {
	s                *rpc.Server
	quickUndoService *quickundo.QuickUndoService
	rconClient       *rcon.Rcon
}

func NewRPCHandler(s *rpc.Server, quickUndoService *quickundo.QuickUndoService, rconClient *rcon.Rcon) *RPCHandler {
	return &RPCHandler{
		s:                s,
		quickUndoService: quickUndoService,
		rconClient:       rconClient,
	}
}

func (h *RPCHandler) HandleGameStop(ctx context.Context, req *rpc.AbstractRequest) error {
	if err := h.rconClient.Stop(); err != nil {
		return err
	}

	exterior.DispatchEvent(ctx, runner.Event{
		Type: runner.EventStatus,
		Status: &runner.StatusExtra{
			EventCode: entity.EventStopping,
		},
	})

	return nil
}

func (h *RPCHandler) HandleSnapshotCreate(ctx context.Context, req *rpc.AbstractRequest) error {
	var input types.SnapshotInput
	if err := req.Bind(&input); err != nil {
		return err
	}

	go func() {
		if err := h.rconClient.SaveAll(); err != nil {
			slog.Error(fmt.Sprintf("Failed to run save-all: %s", err))
			return
		}

		if err := h.quickUndoService.CreateSnapshot(ctx, input.Slot); err != nil {
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

		if err := h.quickUndoService.RestartWithSnapshot(ctx, input.Slot); err != nil {
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
