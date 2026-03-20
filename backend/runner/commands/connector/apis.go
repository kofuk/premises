package connector

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/kofuk/premises/backend/common/entity/runner"
	"github.com/kofuk/premises/backend/runner/rpc"
)

type RPCHandler struct {
	s        *rpc.Server
	config   *runner.Config
	cancelFn func()
}

func NewRPCHandler(s *rpc.Server, config *runner.Config, cancelFn func()) *RPCHandler {
	return &RPCHandler{
		s:        s,
		config:   config,
		cancelFn: cancelFn,
	}
}

func (h *RPCHandler) HandleProxyOpen(ctx context.Context, req *rpc.AbstractRequest) error {
	var connReq runner.ConnReqInfo
	if err := req.Bind(&connReq); err != nil {
		return err
	}

	slog.InfoContext(ctx, "Handling connection", slog.String("id", connReq.ConnectionID))

	slog.InfoContext(ctx, fmt.Sprintf("Endpoint is %s", connReq.Endpoint))

	proxy := &Proxy{
		ID:       connReq.ConnectionID,
		Endpoint: connReq.Endpoint,
		Cert:     connReq.ServerCert,
	}
	go func() {
		if err := proxy.Run(); err != nil {
			slog.ErrorContext(ctx, "Error handling proxy request", slog.Any("error", err))
		}
	}()
	return nil
}

func (h *RPCHandler) HandleBaseStop(ctx context.Context, req *rpc.AbstractRequest) error {
	h.cancelFn()
	return nil
}

func (h *RPCHandler) Bind() {
	h.s.RegisterNotifyMethod("proxy/open", h.HandleProxyOpen)
	h.s.RegisterNotifyMethod("base/stop", h.HandleBaseStop)
}
