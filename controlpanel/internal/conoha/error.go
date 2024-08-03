package conoha

import (
	"encoding/json"
	"fmt"
	"net/http"
)

const (
	OpCreateServer      = "CreateServer"
	OpListFlavorDetails = "ListFlavorDetails"
	OpGetServerDetail   = "GetServerDetail"
	OpListServerDetails = "ListServerDetails"
	OpStopServer        = "StopServer"
	OpDeleteServer      = "DeleteServer"
	OpCreateToken       = "CreateToken"
	OpListImages        = "ListImages"
	OpDeleteImage       = "DeleteImage"
	OpCreateBootVolume  = "CreateBootVolume"
	OpListVolumes       = "ListVolumes"
	OpRenameVolume      = "RenameVolume"
	OpSaveVolumeImage   = "SaveVolumeImage"
)

type ClientError struct {
	Op  string
	Err error
}

func (err ClientError) Error() string {
	return fmt.Sprintf("client error: op=%s: %s", err.Op, err.Err)
}

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
