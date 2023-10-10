package exterior

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type Message struct {
	Type     string `json:"type"`
	UserData string `json:"user_data"`
}

func SendMessage(msg Message) error {
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
