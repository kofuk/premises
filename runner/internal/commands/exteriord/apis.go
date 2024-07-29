package exteriord

import (
	"log/slog"
	"sync"

	"github.com/kofuk/premises/runner/internal/commands/exteriord/outbound"
	"github.com/kofuk/premises/runner/internal/env"
	"github.com/kofuk/premises/runner/internal/rpc"
	"github.com/kofuk/premises/runner/internal/rpc/types"
)

type RPCHandler struct {
	s        *rpc.Server
	msgChan  chan outbound.OutboundMessage
	states   *StateStore
	m        sync.Mutex
	stopHook []string
	cancelFn func()
}

func NewRPCHandler(s *rpc.Server, msgChan chan outbound.OutboundMessage, states *StateStore, cancelFn func()) *RPCHandler {
	return &RPCHandler{
		s:        s,
		msgChan:  msgChan,
		states:   states,
		cancelFn: cancelFn,
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

func (h *RPCHandler) HandleProcRegisterStopHook(req *rpc.AbstractRequest) error {
	var cmd string
	if err := req.Bind(&cmd); err != nil {
		return err
	}

	h.m.Lock()
	h.stopHook = append(h.stopHook, cmd)
	defer h.m.Unlock()

	return nil
}

func (h *RPCHandler) HandleProcDone(req *rpc.AbstractRequest) error {
	h.m.Lock()
	defer h.m.Unlock()

	for _, cmd := range h.stopHook {
		if err := rpc.NewClient(env.DataPath("rpc@"+cmd)).Notify("base/stop", nil); err != nil {
			slog.Warn("Error calling hook", slog.String("cmd", cmd), slog.Any("error", err))
		}
	}

	h.cancelFn()

	return nil
}

func (h *RPCHandler) Bind() {
	h.s.RegisterNotifyMethod("status/push", h.HandleStatusPush)
	h.s.RegisterNotifyMethod("proc/registerStopHook", h.HandleProcRegisterStopHook)
	h.s.RegisterNotifyMethod("proc/done", h.HandleProcDone)
	h.s.RegisterMethod("state/save", h.HandleStateSet)
	h.s.RegisterMethod("state/get", h.HandleStateGet)
	h.s.RegisterMethod("state/remove", h.HandleStateRemove)
}
