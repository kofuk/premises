package exteriord

import (
	"github.com/kofuk/premises/runner/commands/exteriord/msgrouter"
	"github.com/kofuk/premises/runner/rpc"
)

type RPCHandler struct {
	s  *rpc.Server
	mr *msgrouter.MsgRouter
}

func NewRPCHandler(s *rpc.Server, mr *msgrouter.MsgRouter) *RPCHandler {
	return &RPCHandler{
		s:  s,
		mr: mr,
	}
}

func (h *RPCHandler) HandleStatusPush(req *rpc.AbstractRequest) (any, error) {
	var msg msgrouter.Message
	if err := req.Bind(&msg); err != nil {
		return nil, err
	}

	h.mr.DispatchMessage(msg)

	return "ok", nil
}

func (h *RPCHandler) Bind() {
	h.s.RegisterMethod("status/push", h.HandleStatusPush)
}
