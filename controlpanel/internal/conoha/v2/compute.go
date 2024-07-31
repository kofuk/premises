package conoha

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/kofuk/premises/controlpanel/internal/conoha/v2/apitypes"
)

func convertToAvailableFlavorList(flavors []Flavor) []Flavor {
	var output []Flavor
	for _, flavor := range flavors {
		if !strings.HasPrefix(flavor.Name, "g2l-t-") {
			// "g2w-" is Windows and "g2d-" is Database?

			// "g2l-p-" seems to be private flavor.
			// If we call "POST /servers" with such flavorRef, it will be rejected saying "Invalid flavor specification. This flavor is can not be used for public API."
			continue
		}

		output = append(output, flavor)
	}
	return output
}

type Server struct {
	ID string `json:"id"`
}

type CreateServerInput struct {
	FlavorID     string
	RootVolumeID string
	NameTag      string
	UserData     string
}

type CreateServerOutput struct {
	Server Server `json:"server"`
}

func (c *Client) CreateServer(ctx context.Context, input CreateServerInput) (*CreateServerOutput, error) {
	if err := c.updateToken(ctx); err != nil {
		return nil, err
	}

	var apiInput apitypes.CreateServerInput
	apiInput.Server.FlavorID = input.FlavorID
	apiInput.Server.UserData = input.UserData
	apiInput.Server.MetaData.InstanceNameTag = input.NameTag
	apiInput.Server.BlockDevices = []apitypes.BlockDeviceMapping{{UUID: input.RootVolumeID}}

	req, err := newRequest(ctx, http.MethodPost, c.endpoints.Compute, "servers", c.token, apiInput)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusAccepted {
		return nil, ErrorFrom(resp)
	}

	var output CreateServerOutput
	if err := json.NewDecoder(resp.Body).Decode(&output); err != nil {
		return nil, err
	}

	return &output, nil
}

type Flavor struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	RAM        int    `json:"ram"`
	Disk       int    `json:"disk"`
	VCPUs      int    `json:"vcpus"`
	RXTXFactor int    `json:"rxtx_factor"`
	Disabled   bool   `json:"OS-FLV-DISABLED:disabled"`
	Public     bool   `json:"os-flavor-access:is_public"`
}

type ListFlavorDetailsOutput struct {
	Flavors []Flavor `json:"flavors"`
}

func (c *Client) ListFlavorDetails(ctx context.Context) (*ListFlavorDetailsOutput, error) {
	if err := c.updateToken(ctx); err != nil {
		return nil, err
	}

	req, err := newRequest(ctx, http.MethodGet, c.endpoints.Compute, "flavors/detail", c.token, nil)
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

	var output ListFlavorDetailsOutput
	if err := json.NewDecoder(resp.Body).Decode(&output); err != nil {
		return nil, err
	}

	output.Flavors = convertToAvailableFlavorList(output.Flavors)
	return &output, nil
}

type ServerDetail struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Status    string `json:"status"`
	Addresses map[string][]struct {
		Addr    string `json:"addr"`
		Version int    `json:"version"`
	} `json:"addresses"`
	Metadata struct {
		InstanceNameTag string `json:"instance_name_tag"`
	} `json:"metadata"`
	Volumes []struct {
		ID string `json:"id"`
	} `json:"os-extended-volumes:volumes_attached"`
}

type GetServerDetailInput struct {
	ServerID string
}

type GetServerDetailOutput struct {
	Server ServerDetail `json:"server"`
}

func (c *Client) GetServerDetail(ctx context.Context, input GetServerDetailInput) (*GetServerDetailOutput, error) {
	if err := c.updateToken(ctx); err != nil {
		return nil, err
	}

	req, err := newRequest(ctx, http.MethodGet, c.endpoints.Compute, "servers/"+input.ServerID, c.token, nil)
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

	var output GetServerDetailOutput
	if err := json.NewDecoder(resp.Body).Decode(&output); err != nil {
		return nil, err
	}

	return &output, nil
}

type ListServerDetailsOutput struct {
	Servers []ServerDetail `json:"servers"`
}

func (c *Client) ListServerDetails(ctx context.Context) (*ListServerDetailsOutput, error) {
	if err := c.updateToken(ctx); err != nil {
		return nil, err
	}

	req, err := newRequest(ctx, http.MethodGet, c.endpoints.Compute, "servers/detail", c.token, nil)
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

	var output ListServerDetailsOutput
	if err := json.NewDecoder(resp.Body).Decode(&output); err != nil {
		return nil, err
	}

	return &output, nil
}

type StopServerInput struct {
	ServerID string
}

func (c *Client) StopServer(ctx context.Context, input StopServerInput) error {
	if err := c.updateToken(ctx); err != nil {
		return err
	}

	req, err := newRequest(ctx, http.MethodPost, c.endpoints.Compute, "servers/"+input.ServerID+"/action", c.token, struct {
		V *interface{} `json:"os-stop"`
	}{})
	if err != nil {
		return err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusAccepted {
		return ErrorFrom(resp)
	}

	drainBody(resp.Body)

	return nil
}

type DeleteServerInput struct {
	ServerID string
}

func (c *Client) DeleteServer(ctx context.Context, input DeleteServerInput) error {
	if err := c.updateToken(ctx); err != nil {
		return err
	}

	req, err := newRequest(ctx, http.MethodDelete, c.endpoints.Compute, "servers/"+input.ServerID, c.token, nil)
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
