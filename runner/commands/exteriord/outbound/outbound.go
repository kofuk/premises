package outbound

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/kofuk/premises/common/entity/runner"
	"github.com/kofuk/premises/runner/rpc"
	"github.com/kofuk/premises/runner/rpc/types"
)

type ActionMapper func(action runner.Action) error

type OutboundMessage struct {
	Dispatch bool         `json:"dispatch"`
	Event    runner.Event `json:"event"`
}

type Server struct {
	addr          string
	authKey       string
	msgChan       chan OutboundMessage
	actionMappers map[runner.ActionType]ActionMapper
}

func (s *Server) HandleActionStop(action runner.Action) error {
	return rpc.ToLauncher.Call("game/stop", nil, nil)
}

func (s *Server) HandleActionSnapshot(action runner.Action) error {
	if action.Snapshot == nil {
		return errors.New("Missing snapshot config")
	}

	return rpc.ToLauncher.Call("snapshot/create", types.SnapshotInput{
		Slot:  action.Snapshot.Slot,
		Actor: action.Actor,
	}, nil)
}

func (s *Server) HandleActionUndo(action runner.Action) error {
	if action.Snapshot == nil {
		return errors.New("Missing snapshot config")
	}

	return rpc.ToLauncher.Call("snapshot/undo", types.SnapshotInput{
		Slot:  action.Snapshot.Slot,
		Actor: action.Actor,
	}, nil)
}

func (s *Server) HandleActionReconfigure(action runner.Action) error {
	if action.Config == nil {
		return errors.New("Missing config")
	}

	return rpc.ToLauncher.Call("game/reconfigure", action.Config, nil)
}

func NewServer(addr string, authKey string, msgChan chan OutboundMessage) *Server {
	s := &Server{
		addr:          addr,
		authKey:       authKey,
		msgChan:       msgChan,
		actionMappers: make(map[runner.ActionType]ActionMapper),
	}

	s.actionMappers[runner.ActionStop] = s.HandleActionStop
	s.actionMappers[runner.ActionSnapshot] = s.HandleActionSnapshot
	s.actionMappers[runner.ActionUndo] = s.HandleActionUndo
	s.actionMappers[runner.ActionReconfigure] = s.HandleActionReconfigure

	return s
}

func (s *Server) HandleMonitor() {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	buf := bytes.NewBuffer(nil)

	sendStatus := func() {
		req, err := http.NewRequest(http.MethodPost, s.addr+"/_runner/push-status", buf)
		if err != nil {
			slog.Error("Error creating request", slog.Any("error", err))
			return
		}
		req.Header.Set("Authorization", s.authKey)
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

		case msg, ok := <-s.msgChan:
			if !ok {
				break out
			}

			json, err := json.Marshal(msg.Event)
			if err != nil {
				slog.Error("Unabel to marshal event data", slog.Any("error", err))
				continue
			}

			buf.Write(json)
			buf.WriteByte(0)

			if msg.Dispatch {
				sendStatus()
			}
		}
	}

	slog.Error("BUG: client channel has been closed")
}

func (s *Server) PollAction() {
	for {
		req, err := http.NewRequest(http.MethodGet, s.addr+"/_runner/poll-action", nil)
		if err != nil {
			slog.Error("Error creating request", slog.Any("error", err))
			continue
		}
		req.Header.Set("Authorization", s.authKey)

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			slog.Error("Error polling action", slog.Any("error", err))

			time.Sleep(5 * time.Second)
			continue
		}

		defer io.Copy(io.Discard, req.Body)

		var action runner.Action
		dec := json.NewDecoder(resp.Body)
		if err := dec.Decode(&action); err != nil {
			slog.Error("Error decoding request", slog.Any("error", err))
			continue
		}

		mapper, ok := s.actionMappers[action.Type]
		if !ok {
			slog.Error(fmt.Sprintf("Unknown action: %s", action.Type), slog.Any("error", err))
			continue
		}

		go func() {
			// Handle action asynchronously
			if err := mapper(action); err != nil {
				slog.Error(fmt.Sprintf("Error occurred in action mapper: %s", action.Type), slog.Any("error", err))
			}
		}()
	}
}

func (s *Server) Start() {
	go s.PollAction()
	s.HandleMonitor()
}
