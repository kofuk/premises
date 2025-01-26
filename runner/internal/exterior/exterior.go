package exterior

import (
	"context"
	"log/slog"

	"github.com/kofuk/premises/internal/entity/runner"
	potel "github.com/kofuk/premises/internal/otel"
	"github.com/kofuk/premises/runner/internal/rpc"
	"github.com/kofuk/premises/runner/internal/rpc/types"
)

func sendEvent(event runner.Event, dispatch bool) error {
	slog.Debug("Sending message...", slog.Any("data", event))
	return rpc.ToExteriord.Notify("status/push", types.EventInput{
		Dispatch: dispatch,
		Event:    event,
	})
}

// Send status message
func SendEvent(ctx context.Context, event runner.Event) {
	event.Metadata.Traceparent = potel.TraceContextFromContext(ctx)

	if err := sendEvent(event, false); err != nil {
		slog.Error("Unable to send message", slog.Any("error", err))
	}
}

// Same as `SendMessage`, but flushes buffer immediately.
func DispatchEvent(ctx context.Context, event runner.Event) {
	event.Metadata.Traceparent = potel.TraceContextFromContext(ctx)

	if err := sendEvent(event, true); err != nil {
		slog.Error("Unable to send message", slog.Any("error", err))
	}
}
