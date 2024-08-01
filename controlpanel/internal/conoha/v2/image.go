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
		return nil, err
	}

	req, err := newRequest(ctx, http.MethodGet, c.endpoints.Image, "images", c.token, nil)
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

	var output ListImagesOutput
	if err := json.NewDecoder(resp.Body).Decode(&output); err != nil {
		return nil, err
	}

	return &output, nil
}

type DeleteImageInput struct {
	ImageID string
}

func (c *Client) DeleteImage(ctx context.Context, input DeleteImageInput) error {
	if err := c.updateToken(ctx); err != nil {
		return err
	}

	req, err := newRequest(ctx, http.MethodDelete, c.endpoints.Image, "images/"+input.ImageID, c.token, nil)
	if err != nil {
		return err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		return ErrorFrom(resp)
	}

	drainBody(resp.Body)

	return nil
}
