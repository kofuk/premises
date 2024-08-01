package conoha

import (
	"context"
	"encoding/json"
	"net/http"
)

type Image struct {
	ID     string `json:"id"`
	Status string `json:"status"`
	Name   string `json:"name"`
}

type ListImagesOutput struct {
	Images []Image `json:"images"`
}

func (c *Client) ListImages(ctx context.Context) (*ListImagesOutput, error) {
	if err := c.updateToken(ctx); err != nil {
		return nil, ClientError{Op: OpListImages, Err: err}
	}

	req, err := newRequest(ctx, http.MethodGet, c.endpoints.Image, "v2/images", c.token, nil)
	if err != nil {
		return nil, ClientError{Op: OpListImages, Err: err}
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, ClientError{Op: OpListImages, Err: ErrorFrom(resp)}
	}

	var output ListImagesOutput
	if err := json.NewDecoder(resp.Body).Decode(&output); err != nil {
		return nil, ClientError{Op: OpListImages, Err: err}
	}

	return &output, nil
}

type DeleteImageInput struct {
	ImageID string
}

func (c *Client) DeleteImage(ctx context.Context, input DeleteImageInput) error {
	if err := c.updateToken(ctx); err != nil {
		return ClientError{Op: OpDeleteImage, Err: err}
	}

	req, err := newRequest(ctx, http.MethodDelete, c.endpoints.Image, "v2/images/"+input.ImageID, c.token, nil)
	if err != nil {
		return ClientError{Op: OpDeleteImage, Err: err}
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return ClientError{Op: OpDeleteImage, Err: err}
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		return ClientError{Op: OpDeleteImage, Err: ErrorFrom(resp)}
	}

	drainBody(resp.Body)

	return nil
}
