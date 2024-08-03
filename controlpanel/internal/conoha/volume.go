package conoha

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/kofuk/premises/controlpanel/internal/conoha/apitypes"
)

const (
	bootVolumeType = "c3j1-ds02-boot"
)

type CreateBootVolumeInput struct {
	Name    string
	ImageID string
}

type CreateBootVolumeOutput struct {
	Volume struct {
		ID     string `json:"id"`
		Name   string `json:"name"`
		Status string `json:"status"`
	} `json:"volume"`
}

func (c *Client) CreateBootVolume(ctx context.Context, input CreateBootVolumeInput) (*CreateBootVolumeOutput, error) {
	if err := c.updateToken(ctx); err != nil {
		return nil, ClientError{Op: OpCreateBootVolume, Err: err}
	}

	var apiInput apitypes.CreateBootVolumeInput
	apiInput.Volume.Size = 100
	apiInput.Volume.Name = input.Name
	apiInput.Volume.VolumeType = bootVolumeType
	apiInput.Volume.ImageID = input.ImageID
	req, err := newRequest(ctx, http.MethodPost, c.endpoints.Volume, "volumes", c.token, apiInput)
	if err != nil {
		return nil, ClientError{Op: OpCreateBootVolume, Err: err}
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, ClientError{Op: OpCreateBootVolume, Err: err}
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusAccepted {
		return nil, ClientError{Op: OpCreateBootVolume, Err: ErrorFrom(resp)}
	}

	var output CreateBootVolumeOutput
	if err := json.NewDecoder(resp.Body).Decode(&output); err != nil {
		return nil, ClientError{Op: OpCreateBootVolume, Err: err}
	}

	return &output, nil
}

type Volume struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type ListVolumesOutput struct {
	Volumes []Volume `json:"volumes"`
}

func (c *Client) ListVolumes(ctx context.Context) (*ListVolumesOutput, error) {
	if err := c.updateToken(ctx); err != nil {
		return nil, ClientError{Op: OpListVolumes, Err: err}
	}

	req, err := newRequest(ctx, http.MethodGet, c.endpoints.Volume, "volumes", c.token, nil)
	if err != nil {
		return nil, ClientError{Op: OpListVolumes, Err: err}
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, ClientError{Op: OpListVolumes, Err: err}
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, ClientError{Op: OpListVolumes, Err: ErrorFrom(resp)}
	}

	var output ListVolumesOutput
	if err := json.NewDecoder(resp.Body).Decode(&output); err != nil {
		return nil, ClientError{Op: OpListVolumes, Err: err}
	}

	return &output, nil
}

type RenameVolumeInput struct {
	VolumeID string
	Name     string
}

func (c *Client) RenameVolume(ctx context.Context, input RenameVolumeInput) error {
	if err := c.updateToken(ctx); err != nil {
		return ClientError{Op: OpRenameVolume, Err: err}
	}

	var apiInput apitypes.RenameVolumeInput
	apiInput.Volume.Name = input.Name

	req, err := newRequest(ctx, http.MethodPut, c.endpoints.Volume, "volumes/"+input.VolumeID, c.token, apiInput)
	if err != nil {
		return ClientError{Op: OpRenameVolume, Err: err}
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return ClientError{Op: OpRenameVolume, Err: err}
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return ClientError{Op: OpRenameVolume, Err: ErrorFrom(resp)}
	}

	drainBody(resp.Body)

	return nil
}

type SaveVolumeImageInput struct {
	VolumeID  string
	ImageName string
}

func (c *Client) SaveVolumeImage(ctx context.Context, input SaveVolumeImageInput) error {
	if err := c.updateToken(ctx); err != nil {
		return ClientError{Op: OpSaveVolumeImage, Err: err}
	}

	var apiInput apitypes.SaveVolumeImageInput
	apiInput.V.ImageName = input.ImageName
	req, err := newRequest(ctx, http.MethodPost, c.endpoints.Volume, "volumes/"+input.VolumeID+"/action", c.token, apiInput)
	if err != nil {
		return ClientError{Op: OpSaveVolumeImage, Err: err}
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return ClientError{Op: OpSaveVolumeImage, Err: err}
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusAccepted {
		return ClientError{Op: OpSaveVolumeImage, Err: ErrorFrom(resp)}
	}

	drainBody(resp.Body)

	return nil
}
