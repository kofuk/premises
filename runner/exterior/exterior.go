package exterior

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"

	"github.com/kofuk/premises/runner/commands/exteriord/msgrouter"
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

	data, _ := json.Marshal(msg)

	resp, err := http.Post("http://127.0.0.1:2000/pushstatus", "application/json", bytes.NewBuffer(data))
	if err != nil {
		return err
	}
	defer io.ReadAll(resp.Body)

	if resp.StatusCode == http.StatusOK {
		return nil
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	slog.Debug("Sending message...Done")

	return fmt.Errorf("Unable to send message to exteriord: %s", string(body))
}

// Send status message
func SendMessage(msgType string, userData any) error {
	return sendMessage(msgType, userData, false)
}

// Same as `SendMessage`, but flushes buffer immediately.
func DispatchMessage(msgType string, userData any) error {
	return sendMessage(msgType, userData, true)
}
