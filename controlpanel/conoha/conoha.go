package conoha

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"path"
	"strings"
	"time"

	"github.com/kofuk/premises/controlpanel/config"
	log "github.com/sirupsen/logrus"
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

func StopVM(ctx context.Context, cfg *config.Config, token, vmID string) error {
	url, err := url.Parse(cfg.Conoha.Services.Compute)
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

	io.Copy(os.Stdout, resp.Body)

	if resp.StatusCode != 202 {
		return fmt.Errorf("Failed to stop the VM: %d", resp.StatusCode)
	}

	return nil
}

type CreateImageReq struct {
	CreateImage struct {
		Name string `json:"image_name"`
	} `json:"os-volume_upload_image"`
}

func CreateImage(ctx context.Context, cfg *config.Config, token, volumeId, imageName string) error {
	var reqBody CreateImageReq
	reqBody.CreateImage.Name = imageName

	url, err := url.Parse(cfg.Conoha.Services.Volume)
	if err != nil {
		return err
	}
	url.Path = path.Join(url.Path, "volumes", volumeId, "action")

	req, err := makeJSONRequest(ctx, http.MethodPost, url.String(), token, reqBody)
	if err != nil {
		return err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusAccepted {
		return fmt.Errorf("Failed to create image: %d", resp.StatusCode)
	}

	return nil
}

func DeleteImage(ctx context.Context, cfg *config.Config, token, imageID string) error {
	url, err := url.Parse(cfg.Conoha.Services.Image)
	if err != nil {
		return err
	}
	url.Path = path.Join(url.Path, "v2/images", imageID)

	req, err := makeRequest(ctx, http.MethodDelete, url.String(), token)
	if err != nil {
		return err
	}

	log.Info("Deleting image...")
	for i := 0; i < 10; i++ {
		var resp *http.Response
		resp, err = http.DefaultClient.Do(req)
		if err != nil {
			log.WithError(err).Error("Failed to send request")
			return err
		}

		log.WithField("status_code", resp.StatusCode).Info("Requested deleting image")

		if resp.StatusCode == 409 {
			time.Sleep(time.Duration(rand.Intn(10)))
			continue
		}
		if resp.StatusCode != 204 {
			return fmt.Errorf("Failed to delete the image: %d", resp.StatusCode)
		}

		break
	}
	log.Info("Deleting image...Done")

	if err != nil {
		return err
	}
	return nil
}

func DeleteVM(ctx context.Context, cfg *config.Config, token, vmID string) error {
	url, err := url.Parse(cfg.Conoha.Services.Compute)
	if err != nil {
		return err
	}
	url.Path = path.Join(url.Path, "servers", vmID)

	var finalErr error = nil

	for i := 0; i < 10; i++ {
		req, err := makeRequest(ctx, http.MethodDelete, url.String(), token)
		if err != nil {
			return err
		}
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return err
		}

		log.WithField("status_code", resp.StatusCode).Info("Requested deleting VM")

		if resp.StatusCode == 204 {
			finalErr = nil
			break
		} else {
			finalErr = fmt.Errorf("Failed to delete the VM: %d", resp.StatusCode)
			time.Sleep(time.Duration(rand.Intn(10)))
		}

		time.Sleep(time.Duration(rand.Intn(10)))
	}

	return finalErr
}

type CreateVMReq struct {
	Server struct {
		ImageRef  string `json:"imageRef"`
		FlavorRef string `json:"flavorRef"`
		UserData  string `json:"user_data"`
		MetaData  struct {
			InstanceNameTag string `json:"instance_name_tag"`
		} `json:"metadata"`
		SecurityGroups []struct {
			Name string `json:"name"`
		} `json:"security_groups"`
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

func CreateVM(ctx context.Context, cfg *config.Config, nameTag, token, imageRef, flavorRef string, startupScript []byte) (string, error) {
	var reqBody CreateVMReq
	reqBody.Server.ImageRef = imageRef
	reqBody.Server.FlavorRef = flavorRef
	reqBody.Server.UserData = string(base64Encode(startupScript))
	reqBody.Server.MetaData.InstanceNameTag = nameTag
	reqBody.Server.SecurityGroups = []struct {
		Name string `json:"name"`
	}{{nameTag}}

	url, err := url.Parse(cfg.Conoha.Services.Compute)
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
	if resp.StatusCode != 202 {
		return "", fmt.Errorf("Failed to create VM: %d", resp.StatusCode)
	}

	respData, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
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
	url, err := url.Parse(cfg.Conoha.Services.Compute)
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
	if resp.StatusCode == http.StatusNotFound {
		return nil, errors.New("No such VM")
	}
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("Failed to retrieve VM details: %d", resp.StatusCode)
	}

	respData, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
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
	url, err := url.Parse(cfg.Conoha.Services.Compute)
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
	if resp.StatusCode == http.StatusNotFound {
		return nil, errors.New("No such VM")
	}
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("Failed to retrieve VM details: %d", resp.StatusCode)
	}

	respData, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
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

	return nil, errors.New("No such VM")
}

type ImageResp struct {
	Images []struct {
		ID     string `json:"id"`
		Name   string `json:"name"`
		Status string `json:"status"`
	} `json:"images"`
}

func GetImageID(ctx context.Context, cfg *config.Config, token, tag string) (string, string, error) {
	url, err := url.Parse(cfg.Conoha.Services.Image)
	if err != nil {
		return "", "", err
	}
	url.Path = path.Join(url.Path, "v2/images")

	req, err := makeRequest(ctx, http.MethodGet, url.String(), token)
	if err != nil {
		return "", "", err
	}
	query := req.URL.Query()
	query.Add("name", tag)
	req.URL.RawQuery = query.Encode()
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", "", err
	}
	if resp.StatusCode != 200 {
		return "", "", fmt.Errorf("Failed to retrieve image list: %d", resp.StatusCode)
	}

	respData, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", "", err
	}

	var result ImageResp
	if err := json.Unmarshal(respData, &result); err != nil {
		return "", "", err
	}

	if len(result.Images) == 0 {
		return "", "", errors.New("No such image")
	}

	return result.Images[0].ID, result.Images[0].Status, nil
}

type Flavor struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	RAM        int    `json:"ram"`
	Disk       int    `json:"disk"`
	VCPUs      int    `json:"vcpus"`
	RXTXFactor int    `json:"rxtx_factor"`
}

type FlavorsResp struct {
	Flavors []Flavor `json:"flavors"`
}

func GetFlavors(ctx context.Context, cfg *config.Config, token string) ([]Flavor, error) {
	url, err := url.Parse(cfg.Conoha.Services.Compute)
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
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("Failed to retrieve flavor list: %d", resp.StatusCode)
	}

	respData, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var result FlavorsResp
	if err := json.Unmarshal(respData, &result); err != nil {
		return nil, err
	}

	return result.Flavors, nil
}

func FindMatchingFlavor(flavors []Flavor, memSize int) (string, error) {
	var memMatch []Flavor
	for _, fl := range flavors {
		if !strings.HasPrefix(fl.Name, "g2l-") {
			// "g2w-" is Windows and "g2d-" is Database?
			continue
		}
		if fl.RAM == memSize {
			memMatch = append(memMatch, fl)
		}
	}

	if len(memMatch) == 0 {
		return "", errors.New("Matching flavor not found")
	} else {
		return memMatch[0].ID, nil
	}
}

type SecurityGroup struct {
	ID   *string `json:"id,omitempty"`
	Name string  `json:"name"`
}

func GetSecurityGroups(ctx context.Context, cfg *config.Config, token string) ([]SecurityGroup, error) {
	url, err := url.Parse(cfg.Conoha.Services.Network)
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
	url, err := url.Parse(cfg.Conoha.Services.Network)
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
		return "", errors.New("Security group ID shouldn't be nil")
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
	url, err := url.Parse(cfg.Conoha.Services.Network)
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
	if _, err := io.ReadAll(resp.Body); err != nil {
		return err
	}

	if resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("Failed to create security group rule: %d", resp.StatusCode)
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
	auth.Auth.Identity.Password.User.Name = cfg.Conoha.UserName
	auth.Auth.Identity.Password.User.Password = cfg.Conoha.Password
	auth.Auth.Scope.Project.ID = cfg.Conoha.TenantID

	url, err := url.Parse(cfg.Conoha.Services.Identity)
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
	if resp.StatusCode != http.StatusCreated {
		return "", "", fmt.Errorf("Authentication failed: %d", resp.StatusCode)
	}
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", "", err
	}
	var ident GetTokenResp
	if err := json.Unmarshal(respBody, &ident); err != nil {
		return "", "", err
	}
	return resp.Header.Get("x-subject-token"), ident.Token.ExpiresAt, nil
}
