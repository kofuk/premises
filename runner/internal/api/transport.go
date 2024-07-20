package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/kofuk/premises/internal/entity/web"
)

type APITransport struct {
	httpClient *http.Client
	authKey    string
}

type TransportError struct {
	Code int
}

func (e *TransportError) Error() string {
	return fmt.Sprintf("error code: %d", e.Code)
}

func (xp *APITransport) Request(ctx context.Context, method string, url string, body any) ([]byte, error) {
	var reqBody io.Reader
	contentType := "application/octet-stream"

	switch body := body.(type) {
	case nil:
		reqBody = nil
	case []byte:
		reqBody = bytes.NewReader(body)
	default:
		reqData, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		reqBody = bytes.NewBuffer(reqData)
		contentType = "application/json"
	}

	req, err := http.NewRequestWithContext(ctx, method, url, reqBody)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", xp.authKey)
	req.Header.Set("Content-Type", contentType)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}

	var respData web.GenericResponse
	if err := json.NewDecoder(resp.Body).Decode(&respData); err != nil {
		return nil, err
	}

	if !respData.Success {
		return nil, &TransportError{Code: int(respData.ErrorCode)}
	}

	return respData.Data, nil
}
