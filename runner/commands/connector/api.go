package connector

import (
	"log/slog"
	"net"
	"net/url"

	"github.com/kofuk/premises/internal/entity/runner"
	"github.com/kofuk/premises/runner/rpc"
)

type RPCHandler struct {
	s      *rpc.Server
	config *runner.Config
}

func NewRPCHandler(s *rpc.Server, config *runner.Config) *RPCHandler {
	return &RPCHandler{
		s:      s,
		config: config,
	}
}

func (h *RPCHandler) HandleProxyOpen(req *rpc.AbstractRequest) error {
	var connReq runner.ConnReqInfo
	if err := req.Bind(&connReq); err != nil {
		return err
	}

	slog.Info("handling connection", slog.String("id", connReq.ConnectionID))

	url, err := url.Parse(h.config.ControlPanel)
	if err != nil {
		return err
	}

	endpoint := ""
	if host, _, err := net.SplitHostPort(url.Host); err != nil {
		endpoint = url.Host + ":25530"
	} else {
		endpoint = host + ":25530"
	}

	proxy := &Proxy{
		ID:       connReq.ConnectionID,
		Endpoint: endpoint,
		Cert:     connReq.ServerCert,
	}
	return proxy.Start()
}

func (h *RPCHandler) Bind() {
	h.s.RegisterNotifyMethod("proxy/open", h.HandleProxyOpen)
}
