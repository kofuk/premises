package outbound

import (
	"bytes"
	"crypto/subtle"
	"io"
	"log/slog"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"time"

	"github.com/kofuk/premises/runner/commands/exteriord/msgrouter"
)

type Server struct {
	addr      string
	authKey   string
	msgRouter *msgrouter.MsgRouter
}

func NewServer(addr string, authKey string, msgRouter *msgrouter.MsgRouter) *Server {
	return &Server{
		addr:      addr,
		authKey:   authKey,
		msgRouter: msgRouter,
	}
}

func (self *Server) HandleMonitor() {
	client := self.msgRouter.Subscribe(msgrouter.NotifyLatest("serverStatus"))
	defer self.msgRouter.Unsubscribe(client)

	for {
		for msg := range client.C {
			req, err := http.NewRequest(http.MethodPost, self.addr+"/_runner/push-status", bytes.NewBuffer([]byte(msg.UserData)))
			if err != nil {
				slog.Error("Error creating request", slog.Any("error", err))
				continue
			}
			req.Header.Set("Authorization", self.authKey)
			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				slog.Error("Error writing status", slog.Any("error", err))
				continue
			}
			io.ReadAll(resp.Body)
		}
	}
}

func (self *Server) HandleProxy(w http.ResponseWriter, r *http.Request) {
	if subtle.ConstantTimeCompare([]byte(r.Header.Get("X-Auth-Key")), []byte(self.authKey)) == 0 {
		slog.Error("Connection is closed because it has no valid auth key")
		w.WriteHeader(http.StatusForbidden)
		return
	}

	remoteUrl, err := url.Parse("http://127.0.0.1:9000")
	if err != nil {
		slog.Error("[BUG] Unable to parse remote url", slog.Any("error", err))
		os.Exit(1)
	}

	httputil.NewSingleHostReverseProxy(remoteUrl).ServeHTTP(w, r)
}

func (self *Server) PollAction() {
	for {
		req, err := http.NewRequest(http.MethodGet, self.addr+"/_runner/poll-action", nil)
		if err != nil {
			slog.Error("Error creating request", slog.Any("error", err))
			continue
		}
		req.Header.Set("Authorization", self.authKey)

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			slog.Error("Error polling action", slog.Any("error", err))

			time.Sleep(5 * time.Second)
			continue
		}

		data, err := io.ReadAll(resp.Body)
		if err != nil {
			slog.Error("Error reading body", slog.Any("error", err))
			continue
		}

		resp, err = http.Post("http://127.0.0.1:9000/", "application/json", bytes.NewBuffer(data))
		if err != nil {
			slog.Error("Error forwarding action", slog.Any("error", err))
			continue
		}
		io.ReadAll(resp.Body)
	}
}

func (self *Server) Start() {
	go self.PollAction()
	self.HandleMonitor()
}
