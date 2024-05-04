package outbound

import (
	"bytes"
	"io"
	"log/slog"
	"net/http"
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

	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	buf := bytes.NewBuffer(nil)

	sendStatus := func() {
		req, err := http.NewRequest(http.MethodPost, self.addr+"/_runner/push-status", buf)
		if err != nil {
			slog.Error("Error creating request", slog.Any("error", err))
			return
		}
		req.Header.Set("Authorization", self.authKey)
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			slog.Error("Error writing status", slog.Any("error", err))
			return
		}
		io.Copy(io.Discard, resp.Body)

		buf.Reset()
	}

out:
	for {
		select {
		case <-ticker.C:
			if buf.Len() == 0 {
				// If there's no data, don't send message.
				continue out
			}

			sendStatus()

		case msg, ok := <-client.C:
			if !ok {
				break out
			}
			buf.Write([]byte(msg.UserData))
			buf.WriteByte(0)

			if msg.Dispatch {
				sendStatus()
			}
		}
	}

	slog.Error("BUG: client channel has been closed")
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
		io.Copy(io.Discard, resp.Body)
	}
}

func (self *Server) Start() {
	go self.PollAction()
	self.HandleMonitor()
}
