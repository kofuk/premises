package conoha

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/kofuk/premises/controlpanel/internal/conoha/v2/apitypes"
)

type GetTokenInput struct {
	User     string
	Password string
	TenandID string
}

type GetTokenOutput struct {
	Token     string
	ExpiresAt time.Time
}

func (c *Client) CreateToken(ctx context.Context, input GetTokenInput) (*GetTokenOutput, error) {
	var apiInput apitypes.GetTokenInput
	apiInput.Auth.Identity.Methods = []string{"password"}
	apiInput.Auth.Identity.Password.User.Name = c.identity.User
	apiInput.Auth.Identity.Password.User.Password = c.identity.Password
	apiInput.Auth.Scope.Project.ID = c.identity.TenandID

	req, err := newRequest(ctx, http.MethodPost, c.endpoints.Identity, "auth/tokens", "", apiInput)
	if err != nil {
		return nil, ClientError{Op: OpCreateToken, Err: err}
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return nil, ClientError{Op: OpCreateToken, Err: ErrorFrom(resp)}
	}

	var output apitypes.GetTokenOutput
	if err := json.NewDecoder(resp.Body).Decode(&output); err != nil {
		return nil, ClientError{Op: OpCreateToken, Err: err}
	}

	token := resp.Header.Get("x-subject-token")

	expiresAt, err := time.Parse(time.RFC3339, output.Token.ExpiresAt)
	if err != nil {
		return nil, ClientError{Op: OpCreateToken, Err: err}
	}

	return &GetTokenOutput{Token: token, ExpiresAt: expiresAt}, nil

}
