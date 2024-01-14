package exterior

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"

	entity "github.com/kofuk/premises/common/entity/runner"
)

func SendMessage(msgType string, userData any) error {
	slog.Debug("Sending message...", slog.String("type", msgType))

	serializedUserData, err := json.Marshal(userData)
	if err != nil {
		return err
	}

	msg := entity.Message{
		Type:     msgType,
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
