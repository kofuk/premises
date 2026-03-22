package meter

import (
	"context"
	"log/slog"

	"github.com/kofuk/premises/backend/runner/rpc"
	"github.com/kofuk/premises/backend/runner/rpc/types"
)

type RPCHandler struct {
	s            *rpc.Server
	meterService *MeterService
}

func NewRPCHandler(s *rpc.Server, meterService *MeterService) *RPCHandler {
	return &RPCHandler{
		s:            s,
		meterService: meterService,
	}
}

func (h *RPCHandler) HandleRegister(ctx context.Context, req *rpc.AbstractRequest) (any, error) {
	var input types.RegisterMeterTargetInput
	if err := req.Bind(&input); err != nil {
		return struct{}{}, err
	}

	h.meterService.RegisterTarget(input.Pid)

	slog.DebugContext(ctx, "Meter target registered", slog.Int("pid", input.Pid))

	return struct{}{}, nil
}

func (h *RPCHandler) HandleUnregister(ctx context.Context, req *rpc.AbstractRequest) (any, error) {
	var input types.UnregisterMeterTargetInput
	if err := req.Bind(&input); err != nil {
		return struct{}{}, err
	}

	h.meterService.UnregisterTarget(input.Pid)

	slog.DebugContext(ctx, "Meter target unregistered", slog.Int("pid", input.Pid))

	return struct{}{}, nil
}

func (h *RPCHandler) Bind() {
	h.s.RegisterMethod("target/register", h.HandleRegister)
	h.s.RegisterMethod("target/unregister", h.HandleUnregister)
}
