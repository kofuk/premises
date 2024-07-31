package conoha

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"path"
	"time"
)

var (
	headerAuthToken = "X-Auth-Token"
)

type Endpoints struct {
	Identity string
	Compute  string
	Volume   string
}

type Identity struct {
	User     string
	Password string
	TenandID string
}

type Client struct {
	identity   Identity
	endpoints  Endpoints
	token      string
	expiresAt  time.Time
	httpClient *http.Client
}

func NewClient(identity Identity, endpoints Endpoints, httpClient *http.Client) *Client {
	return &Client{
		identity:   identity,
		endpoints:  endpoints,
		httpClient: httpClient,
	}
}

func (c *Client) updateToken(ctx context.Context) error {
	if c.expiresAt.Add(-time.Minute).After(time.Now()) {
		return nil
	}

	token, err := c.CreateToken(ctx, GetTokenInput{
		User:     c.identity.User,
		Password: c.identity.Password,
		TenandID: c.identity.TenandID,
	})
	if err != nil {
		return err
	}

	c.token = token.Token
	c.expiresAt = token.ExpiresAt

	return nil
}

func newRequest(ctx context.Context, method, baseURL, relPath, token string, data any) (*http.Request, error) {
	url, err := url.Parse(baseURL)
	if err != nil {
		return nil, err
	}
	url.Path = path.Join(url.Path, relPath)

	var body io.Reader
	if data != nil {
		json, err := json.Marshal(data)
		if err != nil {
			return nil, err
		}
		body = bytes.NewBuffer(json)
	}

	req, err := http.NewRequestWithContext(ctx, method, url.String(), body)
	if err != nil {
		return nil, err
	}

	if data != nil {
		req.Header.Add("Content-Type", "application/json")
	}

	if token != "" {
		req.Header.Add(headerAuthToken, token)
	}

	return req, nil
}

func drainBody(body io.Reader) {
	io.CopyN(io.Discard, body, 10*1024)
}
