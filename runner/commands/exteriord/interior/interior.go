package interior

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/kofuk/premises/runner/commands/exteriord/msgrouter"
)

type Server struct {
	addr      string
	server    *http.Server
	msgRouter *msgrouter.MsgRouter
}

func NewServer(addr string, msgRouter *msgrouter.MsgRouter) *Server {
	return &Server{
		addr:      addr,
		msgRouter: msgRouter,
	}
}

func (self *Server) HandlePushStatus(w http.ResponseWriter, r *http.Request) {
	bytes, err := io.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	var msg msgrouter.Message
	if err := json.Unmarshal(bytes, &msg); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	self.msgRouter.DispatchMessage(msg)
}

func (self *Server) Start() error {
	mux := http.NewServeMux()
	mux.HandleFunc("/pushstatus", self.HandlePushStatus)

	self.server = &http.Server{Addr: self.addr, Handler: mux}

	return self.server.ListenAndServe()
}

func (self *Server) Close() error {
	return self.server.Close()
}
