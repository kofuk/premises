package conoha

import (
	"encoding/json"
	"fmt"
	"net/http"
)

type APIError struct {
	Code     int    `json:"code"`
	ErrorMsg string `json:"error"`
}

func (err APIError) Error() string {
	return fmt.Sprintf("error calling ConoHa API: %d: %s", err.Code, err.ErrorMsg)
}

func ErrorFrom(resp *http.Response) error {
	var apiErr APIError
	if err := json.NewDecoder(resp.Body).Decode(&apiErr); err != nil {
		return err
	}
	return apiErr
}
