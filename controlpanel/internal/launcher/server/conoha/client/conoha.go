package client

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"path"
	"sync"
	"time"
)

var (
	headerAuthToken = "X-Auth-Token"
	tokenPool       = make(map[string]*token)
	tokenPoolMu     = &sync.Mutex{}
)

type token struct {
	token  string
	expiry time.Time
}

type Endpoints struct {
	Identity string
	Compute  string
	Image    string
	Volume   string
}

type Identity struct {
	User     string
	Password string
	TenantID string
}

type Client struct {
	identity   Identity
	endpoints  Endpoints
	httpClient *http.Client
}

func NewClient(identity Identity, endpoints Endpoints, httpClient *http.Client) *Client {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}

	return &Client{
		identity:   identity,
		endpoints:  endpoints,
		httpClient: httpClient,
	}
}

func (c *Client) getTokenCached(ctx context.Context) (string, error) {
	tokenPoolMu.Lock()
	defer tokenPoolMu.Unlock()

	if tokenPool[c.identity.TenantID].expiry.Add(-time.Minute).After(time.Now()) {
		return tokenPool[c.identity.TenantID].token, nil
	}

	newToken, err := c.CreateToken(ctx, GetTokenInput{
		User:     c.identity.User,
		Password: c.identity.Password,
		TenantID: c.identity.TenantID,
	})
	if err != nil {
		return "", err
	}

	tokenPool[c.identity.TenantID] = &token{
		token:  newToken.Token,
		expiry: newToken.ExpiresAt,
	}

	return newToken.Token, nil
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
