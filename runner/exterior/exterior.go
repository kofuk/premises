package exterior

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/kofuk/premises/runner/exterior/entity"
)

func SendMessage(msgType string, userData any) error {
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

	return fmt.Errorf("Unable to send message to exteriord: %s", string(body))
}
