package conoha

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/kofuk/premises/controlpanel/internal/conoha/v2/apitypes"
)

type ListVolumesOutput struct {
	Volumes []struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	} `json:"volumes"`
}

func (c *Client) ListVolumes(ctx context.Context) (*ListVolumesOutput, error) {
	if err := c.updateToken(ctx); err != nil {
		return nil, err
	}

	req, err := newRequest(ctx, http.MethodGet, c.endpoints.Volume, "volumes", c.token, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, ErrorFrom(resp)
	}

	var output ListVolumesOutput
	if err := json.NewDecoder(resp.Body).Decode(&output); err != nil {
		return nil, err
	}

	return &output, nil
}

type RenameVolumeInput struct {
	VolumeID string
	Name     string
}

func (c *Client) RenameVolume(ctx context.Context, input RenameVolumeInput) error {
	if err := c.updateToken(ctx); err != nil {
		return err
	}

	var apiInput apitypes.RenameVolumeInput
	apiInput.Volume.Name = input.Name

	req, err := newRequest(ctx, http.MethodPut, c.endpoints.Volume, "volumes/"+input.VolumeID, c.token, apiInput)
	if err != nil {
		return err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return ErrorFrom(resp)
	}

	drainBody(resp.Body)

	return nil
}
