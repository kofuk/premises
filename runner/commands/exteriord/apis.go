package exteriord

import (
	"github.com/kofuk/premises/runner/commands/exteriord/outbound"
	"github.com/kofuk/premises/runner/rpc"
	"github.com/kofuk/premises/runner/rpc/types"
)

type RPCHandler struct {
	s       *rpc.Server
	msgChan chan outbound.OutboundMessage
	states  *StateStore
}

func NewRPCHandler(s *rpc.Server, msgChan chan outbound.OutboundMessage, states *StateStore) *RPCHandler {
	return &RPCHandler{
		s:       s,
		msgChan: msgChan,
		states:  states,
	}
}

func (h *RPCHandler) HandleStatusPush(req *rpc.AbstractRequest) error {
	var msg types.EventInput
	if err := req.Bind(&msg); err != nil {
		return err
	}

	h.msgChan <- outbound.OutboundMessage(msg)

	return nil
}

func (h *RPCHandler) HandleStateSet(req *rpc.AbstractRequest) (any, error) {
	var input types.StateSetInput
	if err := req.Bind(&input); err != nil {
		return nil, err
	}

	if err := h.states.Set(input.Key, input.Value); err != nil {
		return nil, err
	}

	return "ok", nil
}

func (h *RPCHandler) HandleStateGet(req *rpc.AbstractRequest) (any, error) {
	var input types.StateGetInput
	if err := req.Bind(&input); err != nil {
		return nil, err
	}

	value, err := h.states.Get(input.Key)
	if err != nil {
		return nil, err
	}

	return value, nil
}

func (h *RPCHandler) HandleStateRemove(req *rpc.AbstractRequest) (any, error) {
	var input types.StateRemoveInput
	if err := req.Bind(&input); err != nil {
		return nil, err
	}

	if err := h.states.Remove(input.Key); err != nil {
		return nil, err
	}

	return "ok", nil
}

func (h *RPCHandler) Bind() {
	h.s.RegisterNotifyMethod("status/push", h.HandleStatusPush)
	h.s.RegisterMethod("state/save", h.HandleStateSet)
	h.s.RegisterMethod("state/get", h.HandleStateGet)
	h.s.RegisterMethod("state/remove", h.HandleStateRemove)
}
