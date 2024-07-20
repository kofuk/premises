package outbound

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/kofuk/premises/internal/entity/runner"
	"github.com/kofuk/premises/runner/internal/api"
	"github.com/kofuk/premises/runner/rpc"
	"github.com/kofuk/premises/runner/rpc/types"
	"golang.org/x/sync/errgroup"
)

type ActionMapper func(action *runner.Action) error

type OutboundMessage struct {
	Dispatch bool         `json:"dispatch"`
	Event    runner.Event `json:"event"`
}

type Server struct {
	client        *api.Client
	msgChan       chan OutboundMessage
	actionMappers map[runner.ActionType]ActionMapper
}

func (s *Server) HandleActionStop(action *runner.Action) error {
	return rpc.ToLauncher.Notify("game/stop", nil)
}

func (s *Server) HandleActionSnapshot(action *runner.Action) error {
	if action.Snapshot == nil {
		return errors.New("missing snapshot config")
	}

	return rpc.ToLauncher.Notify("snapshot/create", types.SnapshotInput{
		Slot:  action.Snapshot.Slot,
		Actor: action.Actor,
	})
}

func (s *Server) HandleActionUndo(action *runner.Action) error {
	if action.Snapshot == nil {
		return errors.New("missing snapshot config")
	}

	return rpc.ToLauncher.Notify("snapshot/undo", types.SnapshotInput{
		Slot:  action.Snapshot.Slot,
		Actor: action.Actor,
	})
}

func (s *Server) HandleActionReconfigure(action *runner.Action) error {
	if action.Config == nil {
		return errors.New("missing config")
	}

	return rpc.ToLauncher.Notify("game/reconfigure", action.Config)
}

func (s *Server) HandleActionConnRequest(action *runner.Action) error {
	if action.ConnReq == nil {
		return errors.New("missing request info")
	}

	return rpc.ToConnector.Notify("proxy/open", action.ConnReq)
}

func NewServer(addr string, authKey string, msgChan chan OutboundMessage) *Server {
	s := &Server{
		client:        api.New(addr, authKey, http.DefaultClient),
		msgChan:       msgChan,
		actionMappers: make(map[runner.ActionType]ActionMapper),
	}

	s.actionMappers[runner.ActionStop] = s.HandleActionStop
	s.actionMappers[runner.ActionSnapshot] = s.HandleActionSnapshot
	s.actionMappers[runner.ActionUndo] = s.HandleActionUndo
	s.actionMappers[runner.ActionReconfigure] = s.HandleActionReconfigure
	s.actionMappers[runner.ActionConnReq] = s.HandleActionConnRequest

	return s
}

func (s *Server) HandleMonitor(ctx context.Context) {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	buf := bytes.NewBuffer(nil)

	sendStatus := func() {
		if err := s.client.PostStatus(ctx, buf.Bytes()); err != nil {
			slog.Error("Error writing status", slog.Any("error", err))
			return
		}

		buf.Reset()
	}

out:
	for {
		select {
		case <-ctx.Done():
			return

		case <-ticker.C:
			if buf.Len() == 0 {
				// If there's no data, don't send message.
				continue out
			}

			sendStatus()

		case msg, ok := <-s.msgChan:
			if !ok {
				slog.Error("BUG: client channel has been closed")
				return
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
}

func (s *Server) PollAction(ctx context.Context) {
	var eg errgroup.Group
	defer eg.Wait()

	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		action, err := s.client.PollAction(ctx)
		if err != nil {
			slog.Error("Error polling action", slog.Any("error", err))

			time.Sleep(5 * time.Second)
			continue
		}

		mapper, ok := s.actionMappers[action.Type]
		if !ok {
			slog.Error(fmt.Sprintf("Unknown action: %s", action.Type), slog.Any("error", err))
			continue
		}

		eg.Go(func() error {
			// Handle action asynchronously
			if err := mapper(action); err != nil {
				slog.Error(fmt.Sprintf("Error occurred in action mapper: %s", action.Type), slog.Any("error", err))
			}
			return nil
		})
	}
}

func (s *Server) Start(ctx context.Context) {
	go s.PollAction(ctx)
	s.HandleMonitor(ctx)
}
