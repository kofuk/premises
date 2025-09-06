package client

//go:generate go tool stringer -type=Operation -output=operation_string.go

import (
	"encoding/json"
	"fmt"
	"net/http"
)

type Operation int

const (
	OpCreateServer Operation = iota
	OpListFlavorDetails
	OpGetServerDetail
	OpListServerDetails
	OpStopServer
	OpDeleteServer
	OpCreateToken
	OpListImages
	OpDeleteImage
	OpCreateBootVolume
	OpListVolumes
	OpRenameVolume
	OpSaveVolumeImage
)

type ClientError struct {
	Op  Operation
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
