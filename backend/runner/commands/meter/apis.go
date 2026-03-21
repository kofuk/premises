package meter

import (
	"context"

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

func (h *RPCHandler) HandleRegister(ctx context.Context, req *rpc.AbstractRequest) error {
	var input types.RegisterMeterTargetInput
	if err := req.Bind(&input); err != nil {
		return err
	}

	h.meterService.RegisterTarget(input.Pid)

	return nil
}

func (h *RPCHandler) HandleUnregister(ctx context.Context, req *rpc.AbstractRequest) error {
	var input types.UnregisterMeterTargetInput
	if err := req.Bind(&input); err != nil {
		return err
	}

	h.meterService.UnregisterTarget(input.Pid)

	return nil
}

func (h *RPCHandler) Bind() {
	h.s.RegisterNotifyMethod("target/register", h.HandleRegister)
	h.s.RegisterNotifyMethod("target/unregister", h.HandleUnregister)
}
