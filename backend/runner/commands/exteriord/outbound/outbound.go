package outbound

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/kofuk/premises/backend/common/entity/runner"
	"github.com/kofuk/premises/backend/common/otel"
	"github.com/kofuk/premises/backend/runner/api"
	"github.com/kofuk/premises/backend/runner/rpc"
	"github.com/kofuk/premises/backend/runner/rpc/types"
	"golang.org/x/sync/errgroup"
)

type ActionMapper func(ctx context.Context, action *runner.Action) error

type OutboundMessage struct {
	Dispatch bool         `json:"dispatch"`
	Event    runner.Event `json:"event"`
}

type Server struct {
	client        *api.Client
	msgChan       chan OutboundMessage
	actionMappers map[runner.ActionType]ActionMapper
}

func (s *Server) HandleActionStop(ctx context.Context, action *runner.Action) error {
	return rpc.ToLauncher.Notify(ctx, "game/stop", nil)
}

func (s *Server) HandleActionSnapshot(ctx context.Context, action *runner.Action) error {
	if action.Snapshot == nil {
		return errors.New("missing snapshot config")
	}

	return rpc.ToLauncher.Notify(ctx, "snapshot/create", types.SnapshotInput{
		Slot:  action.Snapshot.Slot,
		Actor: action.Actor,
	})
}

func (s *Server) HandleActionUndo(ctx context.Context, action *runner.Action) error {
	if action.Snapshot == nil {
		return errors.New("missing snapshot config")
	}

	return rpc.ToLauncher.Notify(ctx, "snapshot/undo", types.SnapshotInput{
		Slot:  action.Snapshot.Slot,
		Actor: action.Actor,
	})
}

func (s *Server) HandleActionReconfigure(ctx context.Context, action *runner.Action) error {
	if action.Config == nil {
		return errors.New("missing config")
	}

	return rpc.ToLauncher.Notify(ctx, "game/reconfigure", action.Config)
}

func (s *Server) HandleActionConnRequest(ctx context.Context, action *runner.Action) error {
	if action.ConnReq == nil {
		return errors.New("missing request info")
	}

	return rpc.ToConnector.Notify(ctx, "proxy/open", action.ConnReq)
}

func NewServer(addr string, authKey string, msgChan chan OutboundMessage) *Server {
	s := &Server{
		client:        api.NewClient(addr, authKey, http.DefaultClient),
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
		defer buf.Reset()

		if err := s.client.PostStatus(ctx, buf.Bytes()); err != nil {
			slog.ErrorContext(ctx, "Error writing status", slog.Any("error", err))
			return
		}
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
				slog.ErrorContext(ctx, "BUG: client channel has been closed")
				return
			}

			json, err := json.Marshal(msg.Event)
			if err != nil {
				slog.ErrorContext(ctx, "Unable to marshal event data", slog.Any("error", err))
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
			slog.ErrorContext(ctx, "Error polling action", slog.Any("error", err))

			time.Sleep(5 * time.Second)
			continue
		}

		mapper, ok := s.actionMappers[action.Type]
		if !ok {
			slog.ErrorContext(ctx, "Unknown action", slog.String("action", action.Type.String()), slog.Any("error", err))
			continue
		}

		eg.Go(func() error {
			ctx := otel.ContextFromTraceContext(context.Background(), action.Metadata.Traceparent)

			// Handle action asynchronously
			if err := mapper(ctx, action); err != nil {
				slog.ErrorContext(ctx, "Error occurred in action mapper", slog.String("action", action.Type.String()), slog.Any("error", err))
			}
			return nil
		})
	}
}

func (s *Server) Start(ctx context.Context) {
	go s.PollAction(ctx)
	s.HandleMonitor(ctx)
}
