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
	metrics  *Metrics
}

func NewRPCHandler(s *rpc.Server, config *runner.Config, cancelFn func(), metrics *Metrics) *RPCHandler {
	return &RPCHandler{
		s:        s,
		config:   config,
		cancelFn: cancelFn,
		metrics:  metrics,
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
		Metrics:  h.metrics,
	}
	go func() {
		h.metrics.openCount.Add(ctx, 1)

		if err := proxy.Run(ctx); err != nil {
			slog.ErrorContext(ctx, "Error handling proxy request", slog.Any("error", err))
		}

		h.metrics.closeCount.Add(ctx, 1)
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
