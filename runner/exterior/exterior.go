package exterior

import (
	"encoding/json"
	"log/slog"

	"github.com/kofuk/premises/runner/commands/exteriord/msgrouter"
	"github.com/kofuk/premises/runner/rpc"
)

func sendMessage(msgType string, userData any, dispatch bool) error {
	slog.Debug("Sending message...", slog.String("type", msgType), slog.Any("data", userData))

	serializedUserData, err := json.Marshal(userData)
	if err != nil {
		return err
	}

	msg := msgrouter.Message{
		Type:     msgType,
		Dispatch: dispatch,
		UserData: string(serializedUserData),
	}

	var result string
	return rpc.ToExteriord.Call("status/push", msg, &result)
}

// Send status message
func SendMessage(msgType string, userData any) {
	if err := sendMessage(msgType, userData, false); err != nil {
		slog.Error("Unable to send message", slog.Any("error", err))
	}
}

// Same as `SendMessage`, but flushes buffer immediately.
func DispatchMessage(msgType string, userData any) {
	if err := sendMessage(msgType, userData, true); err != nil {
		slog.Error("Unable to send message", slog.Any("error", err))
	}
}
