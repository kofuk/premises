package conoha

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"path"
	"strings"
	"time"

	"github.com/kofuk/premises/common/retry"
	"github.com/kofuk/premises/controlpanel/config"
)

const (
	headerKeyAuthToken = "X-Auth-Token"
)

func makeJSONRequest(ctx context.Context, method, url, token string, data any) (*http.Request, error) {
	json, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, method, url, bytes.NewBuffer(json))
	if err != nil {
		return nil, err
	}
	req.Header.Add("Content-Type", "application/json")
	if token != "" {
		req.Header.Add(headerKeyAuthToken, token)
	}
	return req, nil
}

func makeRequest(ctx context.Context, method, url, token string) (*http.Request, error) {
	req, err := http.NewRequestWithContext(ctx, method, url, nil)
	if err != nil {
		return nil, err
	}
	if token != "" {
		req.Header.Add(headerKeyAuthToken, token)
	}
	return req, nil
}

type APIError struct {
	Code     int    `json:"code"`
	ErrorMsg string `json:"error"`
}

func ErrorFrom(statusCode int, respData []byte) APIError {
	var result APIError
	if err := json.Unmarshal(respData, &result); err != nil {
		result.Code = statusCode
	}
	return result
}

func (err APIError) Error() string {
	return fmt.Sprintf("APIError: %d: %s", err.Code, err.ErrorMsg)
}

func StopVM(ctx context.Context, cfg *config.Config, token, vmID string) error {
	url, err := url.Parse(cfg.ConohaComputeService)
	if err != nil {
		return err
	}
	url.Path = path.Join(url.Path, "servers", vmID, "action")

	req, err := makeJSONRequest(ctx, http.MethodPost, url.String(), token, struct {
		V *interface{} `json:"os-stop"`
	}{})
	if err != nil {
		return err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}

	respData, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusAccepted {
		return ErrorFrom(resp.StatusCode, respData)
	}

	return nil
}

func DeleteVM(ctx context.Context, cfg *config.Config, token, vmID string) error {
	url, err := url.Parse(cfg.ConohaComputeService)
	if err != nil {
		return err
	}
	url.Path = path.Join(url.Path, "servers", vmID)

	return retry.Retry(func() error {
		req, err := makeRequest(ctx, http.MethodDelete, url.String(), token)
		if err != nil {
			return err
		}
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return err
		}

		slog.Info("Requested deleting VM", slog.Int("status_code", resp.StatusCode))

		respData, err := io.ReadAll(resp.Body)
		if err != nil {
			return err
		}

		if resp.StatusCode == http.StatusNoContent {
			return nil
		}

		return ErrorFrom(resp.StatusCode, respData)
	}, 3*time.Minute)
}

type BlockDeviceMapping struct {
	UUID string `json:"uuid"`
}

type CreateVMReq struct {
	Server struct {
		FlavorRef string `json:"flavorRef"`
		UserData  string `json:"user_data"`
		MetaData  struct {
			InstanceNameTag string `json:"instance_name_tag"`
		} `json:"metadata"`
		SecurityGroups []struct {
			Name string `json:"name"`
		} `json:"security_groups"`
		BlockDevices []BlockDeviceMapping `json:"block_device_mapping_v2"`
	} `json:"server"`
}

type CreateVMResp struct {
	Server struct {
		ID string `json:"id"`
	} `json:"server"`
}

func base64Encode(data []byte) []byte {
	result := make([]byte, base64.RawStdEncoding.EncodedLen(len(data)))
	base64.RawStdEncoding.Encode(result, data)
	return result
}

func CreateVM(ctx context.Context, cfg *config.Config, nameTag, token, volumeId string, flavor Flavor, startupScript []byte) (string, error) {
	var reqBody CreateVMReq
	reqBody.Server.FlavorRef = flavor.ID
	reqBody.Server.UserData = string(base64Encode(startupScript))
	reqBody.Server.MetaData.InstanceNameTag = nameTag
	reqBody.Server.SecurityGroups = []struct {
		Name string `json:"name"`
	}{{nameTag}}
	reqBody.Server.BlockDevices = append(reqBody.Server.BlockDevices, BlockDeviceMapping{UUID: volumeId})

	url, err := url.Parse(cfg.ConohaComputeService)
	if err != nil {
		return "", err
	}
	url.Path = path.Join(url.Path, "servers")

	req, err := makeJSONRequest(ctx, http.MethodPost, url.String(), token, reqBody)
	if err != nil {
		return "", err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}

	respData, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	if resp.StatusCode != http.StatusAccepted {
		return "", ErrorFrom(resp.StatusCode, respData)
	}

	var result CreateVMResp
	if err := json.Unmarshal(respData, &result); err != nil {
		return "", err
	}

	return result.Server.ID, nil
}

type VMDetail struct {
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

type VMDetailResp struct {
	Server VMDetail `json:"server"`
}

func GetVMDetail(ctx context.Context, cfg *config.Config, token, id string) (*VMDetail, error) {
	url, err := url.Parse(cfg.ConohaComputeService)
	if err != nil {
		return nil, err
	}
	url.Path = path.Join(url.Path, "servers", id)

	req, err := makeRequest(ctx, http.MethodGet, url.String(), token)
	if err != nil {
		return nil, err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}

	respData, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode == http.StatusNotFound {
		return nil, errors.New("no such VM")
	}
	if resp.StatusCode != http.StatusOK {
		return nil, ErrorFrom(resp.StatusCode, respData)
	}

	var result VMDetailResp
	if err := json.Unmarshal(respData, &result); err != nil {
		return nil, err
	}

	return &result.Server, nil
}

type VMDetailListResp struct {
	Servers []VMDetail `json:"servers"`
}

type FindVMFunc func(server *VMDetail) bool

func FindByName(name string) FindVMFunc {
	return func(server *VMDetail) bool {
		return server.Metadata.InstanceNameTag == name
	}
}

func FindByIPAddr(ipv4Addr string) FindVMFunc {
	return func(server *VMDetail) bool {
		for _, addresses := range server.Addresses {
			for _, addr := range addresses {
				if addr.Version == 4 && addr.Addr == ipv4Addr {
					return true
				}
			}
		}
		return false
	}
}

func FindVM(ctx context.Context, cfg *config.Config, token string, condition FindVMFunc) (*VMDetail, error) {
	url, err := url.Parse(cfg.ConohaComputeService)
	if err != nil {
		return nil, err
	}
	url.Path = path.Join(url.Path, "servers/detail")

	req, err := makeRequest(ctx, http.MethodGet, url.String(), token)
	if err != nil {
		return nil, err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}

	respData, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode == http.StatusNotFound {
		return nil, errors.New("no such VM")
	}
	if resp.StatusCode != http.StatusOK {
		return nil, ErrorFrom(resp.StatusCode, respData)
	}

	var result VMDetailListResp
	if err := json.Unmarshal(respData, &result); err != nil {
		return nil, err
	}

	for _, instance := range result.Servers {
		if condition(&instance) {
			return &instance, nil
		}
	}

	return nil, errors.New("no such VM")
}

type VolumeResp struct {
	Volumes []struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	} `json:"volumes"`
}

func GetVolumeID(ctx context.Context, cfg *config.Config, token, tag string) (string, error) {
	url, err := url.Parse(cfg.ConohaVolumeService)
	if err != nil {
		return "", err
	}
	url.Path = path.Join(url.Path, "volumes")

	req, err := makeRequest(ctx, http.MethodGet, url.String(), token)
	if err != nil {
		return "", err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}

	respData, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	if resp.StatusCode != http.StatusOK {
		return "", ErrorFrom(resp.StatusCode, respData)
	}

	var result VolumeResp
	if err := json.Unmarshal(respData, &result); err != nil {
		return "", err
	}

	for _, volume := range result.Volumes {
		if volume.Name == tag {
			return volume.ID, nil
		}
	}

	return "", errors.New("volume not found")
}

type VolumeRenameReq struct {
	Volume struct {
		Name string `json:"name"`
	} `json:"volume"`
}

func RenameVolume(ctx context.Context, cfg *config.Config, token, volumeId, name string) error {
	url, err := url.Parse(cfg.ConohaVolumeService)
	if err != nil {
		return err
	}
	url.Path = path.Join(url.Path, "volumes", volumeId)

	reqBody := VolumeRenameReq{}
	reqBody.Volume.Name = name

	req, err := makeJSONRequest(ctx, http.MethodPut, url.String(), token, reqBody)
	if err != nil {
		return err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}

	respData, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK {
		return ErrorFrom(resp.StatusCode, respData)
	}

	return nil
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

type FlavorsResp struct {
	Flavors []Flavor `json:"flavors"`
}

func GetFlavors(ctx context.Context, cfg *config.Config, token string) ([]Flavor, error) {
	url, err := url.Parse(cfg.ConohaComputeService)
	if err != nil {
		return nil, err
	}
	url.Path = path.Join(url.Path, "flavors/detail")

	req, err := makeRequest(ctx, http.MethodGet, url.String(), token)
	if err != nil {
		return nil, err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}

	respData, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, ErrorFrom(resp.StatusCode, respData)
	}

	var flavors FlavorsResp
	if err := json.Unmarshal(respData, &flavors); err != nil {
		return nil, err
	}

	var result []Flavor
	for _, flavor := range flavors.Flavors {
		if flavor.Disabled || !flavor.Public {
			continue
		}
		result = append(result, flavor)
	}

	return result, nil
}

func FindMatchingFlavor(flavors []Flavor, memSize int) (Flavor, error) {
	var memMatch []Flavor
	for _, fl := range flavors {
		if !strings.HasPrefix(fl.Name, "g2l-t-") {
			// "g2w-" is Windows and "g2d-" is Database?

			// "g2l-p-" seems to be private flavor.
			// If we call "POST /servers" with such flavorRef, it will be rejected saying "Invalid flavor specification. This flavor is can not be used for public API."
			continue
		}
		if fl.RAM == memSize {
			memMatch = append(memMatch, fl)
		}
	}

	if len(memMatch) == 0 {
		return Flavor{}, errors.New("matching flavor not found")
	} else {
		return memMatch[0], nil
	}
}

type SecurityGroup struct {
	ID   *string `json:"id,omitempty"`
	Name string  `json:"name"`
}

func GetSecurityGroups(ctx context.Context, cfg *config.Config, token string) ([]SecurityGroup, error) {
	url, err := url.Parse(cfg.ConohaNetworkService)
	if err != nil {
		return nil, err
	}
	url.Path = path.Join(url.Path, "v2.0/security-groups")

	req, err := makeRequest(ctx, http.MethodGet, url.String(), token)
	if err != nil {
		return nil, err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	var sg struct {
		SecurityGroups []SecurityGroup `json:"security_groups"`
	}
	if err := json.Unmarshal(respBody, &sg); err != nil {
		return nil, err
	}

	return sg.SecurityGroups, nil
}

func CreateSecurityGroup(ctx context.Context, cfg *config.Config, token, name string) (string, error) {
	url, err := url.Parse(cfg.ConohaNetworkService)
	if err != nil {
		return "", err
	}
	url.Path = path.Join(url.Path, "v2.0/security-groups")

	req, err := makeJSONRequest(ctx, http.MethodPost, url.String(), token, struct {
		SecurityGroup SecurityGroup `json:"security_group"`
	}{
		SecurityGroup: SecurityGroup{
			Name: name,
		},
	})
	if err != nil {
		return "", err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var sg struct {
		SecurityGroup SecurityGroup `json:"security_group"`
	}
	if err := json.Unmarshal(respBody, &sg); err != nil {
		return "", err
	}

	if sg.SecurityGroup.ID == nil {
		return "", errors.New("security group ID shouldn't be nil")
	}

	return *sg.SecurityGroup.ID, nil
}

type SecurityGroupRule struct {
	SecurityGroupID string `json:"security_group_id"`
	Direction       string `json:"direction"`
	EtherType       string `json:"ethertype"`
	PortRangeMin    string `json:"port_range_min"`
	PortRangeMax    string `json:"port_range_max"`
	Protocol        string `json:"protocol"`
	RemoteIP        string `json:"remote_ip_prefix"`
}

func CreateSecurityGroupRule(ctx context.Context, cfg *config.Config, token string, rule SecurityGroupRule) error {
	url, err := url.Parse(cfg.ConohaNetworkService)
	if err != nil {
		return err
	}
	url.Path = path.Join(url.Path, "v2.0/security-group-rules")

	req, err := makeJSONRequest(ctx, http.MethodPost, url.String(), token, struct {
		SecurityGroupRule SecurityGroupRule `json:"security_group_rule"`
	}{
		SecurityGroupRule: rule,
	})
	if err != nil {
		return err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}

	respData, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusCreated {
		return ErrorFrom(resp.StatusCode, respData)
	}

	return nil
}

type GetTokenReq struct {
	Auth struct {
		Identity struct {
			Methods  []string `json:"methods"`
			Password struct {
				User struct {
					Name     string `json:"name"`
					Password string `json:"password"`
				} `json:"user"`
			} `json:"password"`
		} `json:"identity"`
		Scope struct {
			Project struct {
				ID string `json:"id"`
			} `json:"project"`
		} `json:"scope"`
	} `json:"auth"`
}

type GetTokenResp struct {
	Token struct {
		ExpiresAt string `json:"expires_at"`
	} `json:"token"`
}

func GetToken(ctx context.Context, cfg *config.Config) (string, string, error) {
	var auth GetTokenReq
	auth.Auth.Identity.Methods = append(auth.Auth.Identity.Methods, "password")
	auth.Auth.Identity.Password.User.Name = cfg.ConohaUser
	auth.Auth.Identity.Password.User.Password = cfg.ConohaPassword
	auth.Auth.Scope.Project.ID = cfg.ConohaTenantID

	url, err := url.Parse(cfg.ConohaIdentityService)
	if err != nil {
		return "", "", err
	}
	url.Path = path.Join(url.Path, "auth/tokens")

	req, err := makeJSONRequest(ctx, http.MethodPost, url.String(), "", auth)
	if err != nil {
		return "", "", err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", "", err
	}

	respData, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", "", err
	}

	if resp.StatusCode != http.StatusCreated {
		return "", "", ErrorFrom(resp.StatusCode, respData)
	}
	var ident GetTokenResp
	if err := json.Unmarshal(respData, &ident); err != nil {
		return "", "", err
	}
	return resp.Header.Get("x-subject-token"), ident.Token.ExpiresAt, nil
}
