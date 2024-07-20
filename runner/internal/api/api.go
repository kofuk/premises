package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"

	"github.com/kofuk/premises/internal/entity/runner"
	"github.com/kofuk/premises/internal/entity/web"
)

type Client struct {
	endpoint  string
	transport *APITransport
}

func New(endpoint, authKey string, httpClient *http.Client) *Client {
	return &Client{
		endpoint:  endpoint,
		transport: &APITransport{httpClient: httpClient, authKey: authKey},
	}
}

func buildURL(endpoint, path string) (string, error) {
	url, err := url.Parse(endpoint)
	if err != nil {
		return "", err
	}
	url.Path = path
	return url.String(), nil
}

func (c *Client) CreateWorldDownloadURL(ctx context.Context, worldID string) (*web.CreateWorldDownloadURLResponse, error) {
	req := web.CreateWorldDownloadURLRequest{WorldID: worldID}

	url, err := buildURL(c.endpoint, "/_/world/download-url")
	if err != nil {
		return nil, err
	}

	resp, err := c.transport.Request(ctx, http.MethodPost, url, req)
	if err != nil {
		return nil, err
	}

	var respData web.CreateWorldDownloadURLResponse
	if err := json.Unmarshal(resp, &respData); err != nil {
		return nil, err
	}

	return &respData, nil
}

func (c *Client) CreateWorldUploadURL(ctx context.Context, worldName string) (*web.CreateWorldUploadURLResponse, error) {
	req := web.CreateWorldUploadURLRequest{WorldName: worldName}

	url, err := buildURL(c.endpoint, "/_/world/upload-url")
	if err != nil {
		return nil, err
	}

	resp, err := c.transport.Request(ctx, http.MethodPost, url, req)
	if err != nil {
		return nil, err
	}

	var respData web.CreateWorldUploadURLResponse
	if err := json.Unmarshal(resp, &respData); err != nil {
		return nil, err
	}

	return &respData, nil
}

func (c *Client) GetLatestWorldID(ctx context.Context, worldName string) (*web.GetLatestWorldIDResponse, error) {
	url, err := buildURL(c.endpoint, "/_/world/latest-id/"+worldName)
	if err != nil {
		return nil, err
	}

	resp, err := c.transport.Request(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	var respData web.GetLatestWorldIDResponse
	if err := json.Unmarshal(resp, &respData); err != nil {
		return nil, err
	}

	return &respData, nil
}

func (c *Client) PostStatus(ctx context.Context, statuses []byte) error {
	url, err := buildURL(c.endpoint, "/_/status")
	if err != nil {
		return err
	}

	_, err = c.transport.Request(ctx, http.MethodPost, url, statuses)
	return err
}

func (c *Client) PollAction(ctx context.Context) (*runner.Action, error) {
	url, err := buildURL(c.endpoint, "/_/poll")
	if err != nil {
		return nil, err
	}

	resp, err := c.transport.Request(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	var action runner.Action
	if err := json.Unmarshal(resp, &action); err != nil {
		return nil, err
	}

	return &action, nil
}
