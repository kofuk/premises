package conoha

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/kofuk/premises/home/config"
	log "github.com/sirupsen/logrus"
)

const (
	HeaderKeyAuthToken = "X-Auth-Token"
)

func StopVM(cfg *config.Config, token, vmID string) error {
	url, err := url.Parse(cfg.Conoha.Services.Compute)
	if err != nil {
		return err
	}
	url.Path = path.Join(url.Path, "servers", vmID, "action")

	req, err := http.NewRequest("POST", url.String(), bytes.NewBuffer([]byte("{\"os-stop\": null}")))
	if err != nil {
		return err
	}
	req.Header.Add(HeaderKeyAuthToken, token)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	io.Copy(os.Stdout, resp.Body)

	if resp.StatusCode != 202 {
		return errors.New(fmt.Sprintf("Failed to stop the VM: %d", resp.StatusCode))
	}

	return nil
}

type CreateImageReq struct {
	CreateImage struct {
		Name string `json:"name"`
	} `json:"createImage"`
}

func CreateImage(cfg *config.Config, token, vmID, imageName string) error {
	var reqBody CreateImageReq
	reqBody.CreateImage.Name = imageName

	reqData, err := json.Marshal(reqBody)
	if err != nil {
		return err
	}

	url, err := url.Parse(cfg.Conoha.Services.Compute)
	if err != nil {
		return err
	}
	url.Path = path.Join(url.Path, "servers", vmID, "action")

	req, err := http.NewRequest("POST", url.String(), bytes.NewBuffer(reqData))
	if err != nil {
		return err
	}
	req.Header.Add(HeaderKeyAuthToken, token)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 202 {
		return errors.New(fmt.Sprintf("Failed to create image: %d", resp.StatusCode))
	}

	return nil
}

func DeleteImage(cfg *config.Config, token, imageID string) error {
	url, err := url.Parse(cfg.Conoha.Services.Image)
	if err != nil {
		return err
	}
	url.Path = path.Join(url.Path, "v2/images", imageID)

	req, err := http.NewRequest("DELETE", url.String(), nil)
	if err != nil {
		return err
	}
	req.Header.Add(HeaderKeyAuthToken, token)

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
			return errors.New(fmt.Sprintf("Failed to delete the image: %d", resp.StatusCode))
		}

		break
	}
	log.Info("Deleting image...Done")

	if err != nil {
		return err
	}
	return nil
}

func DeleteVM(cfg *config.Config, token, vmID string) error {
	url, err := url.Parse(cfg.Conoha.Services.Compute)
	if err != nil {
		return err
	}
	url.Path = path.Join(url.Path, "servers", vmID)

	req, err := http.NewRequest("DELETE", url.String(), nil)
	if err != nil {
		return err
	}
	req.Header.Add(HeaderKeyAuthToken, token)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 204 {
		return errors.New(fmt.Sprintf("Failed to delete the VM: %d", resp.StatusCode))
	}

	return nil
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

func CreateVM(cfg *config.Config, token, imageRef, flavorRef, encodedStartupScript string) (string, error) {
	var reqBody CreateVMReq
	reqBody.Server.ImageRef = imageRef
	reqBody.Server.FlavorRef = flavorRef
	reqBody.Server.UserData = encodedStartupScript
	reqBody.Server.MetaData.InstanceNameTag = "mc-premises"
	reqBody.Server.SecurityGroups = []struct {
		Name string `json:"name"`
	}{{"default"}, {"gncs-ipv4-all"}, {"gncs-ipv6-all"}}
	reqData, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}

	url, err := url.Parse(cfg.Conoha.Services.Compute)
	if err != nil {
		return "", err
	}
	url.Path = path.Join(url.Path, "servers")

	req, err := http.NewRequest("POST", url.String(), bytes.NewBuffer(reqData))
	if err != nil {
		return "", err
	}
	req.Header.Add(HeaderKeyAuthToken, token)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 202 {
		return "", errors.New(fmt.Sprintf("Failed to create VM: %d", resp.StatusCode))
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
}

type VMDetailResp struct {
	Servers []VMDetail `json:"servers"`
}

func GetVMDetail(cfg *config.Config, token, name string) (*VMDetail, error) {
	url, err := url.Parse(cfg.Conoha.Services.Compute)
	if err != nil {
		return nil, err
	}
	url.Path = path.Join(url.Path, "servers", "detail")

	req, err := http.NewRequest("GET", url.String(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Add(HeaderKeyAuthToken, token)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, errors.New(fmt.Sprintf("Failed to retrieve VM details: %d", resp.StatusCode))
	}

	respData, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var result VMDetailResp
	if err := json.Unmarshal(respData, &result); err != nil {
		return nil, err
	}

	for _, instance := range result.Servers {
		if instance.Metadata.InstanceNameTag == name {
			return &instance, nil
		}
	}

	return nil, errors.New("No such VM")
}

func (d *VMDetail) GetIPAddress(ipVersion int) string {
	for _, iface := range d.Addresses {
		for _, addr := range iface {
			if addr.Version == ipVersion {
				return addr.Addr
			}
		}
	}
	return ""
}

type ImageResp struct {
	Images []struct {
		ID     string `json:"id"`
		Name   string `json:"name"`
		Status string `json:"status"`
	} `json:"images"`
}

func GetImageID(cfg *config.Config, token, tag string) (string, string, error) {
	url, err := url.Parse(cfg.Conoha.Services.Image)
	if err != nil {
		return "", "", err
	}
	url.Path = path.Join(url.Path, "v2/images")

	req, err := http.NewRequest("GET", url.String(), nil)
	if err != nil {
		return "", "", err
	}
	query := req.URL.Query()
	query.Add("name", tag)
	req.URL.RawQuery = query.Encode()
	req.Header.Add(HeaderKeyAuthToken, token)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return "", "", errors.New(fmt.Sprintf("Failed to retrieve image list: %d", resp.StatusCode))
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

type FlavorsResp struct {
	Flavors []struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	} `json:"flavors"`
}

func GetFlavors(cfg *config.Config, token string) (*FlavorsResp, error) {
	url, err := url.Parse(cfg.Conoha.Services.Compute)
	if err != nil {
		return nil, err
	}
	url.Path = path.Join(url.Path, "flavors")

	req, err := http.NewRequest("GET", url.String(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Add(HeaderKeyAuthToken, token)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, errors.New(fmt.Sprintf("Failed to retrieve flavor list: %d", resp.StatusCode))
	}

	respData, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var result FlavorsResp
	if err := json.Unmarshal(respData, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

var unsupportedFlavorError = errors.New("Unsupported flavor name")

func getSpecFromFlavorName(name string) (int, int, int, error) {
	if name[:3] != "g-c" {
		return 0, 0, 0, unsupportedFlavorError
	}
	var strFields [3]strings.Builder
	curField := 0
	name = name[3:]
	for _, c := range name {
		if curField == 0 && c == 'm' {
			curField++
		} else if curField == 1 && c == 'd' {
			curField++
		} else if '0' <= c && c <= '9' {
			strFields[curField].WriteRune(c)
		} else {
			return 0, 0, 0, unsupportedFlavorError
		}
	}

	var fields [3]int

	for i, f := range strFields {
		if f.Len() == 0 {
			return 0, 0, 0, unsupportedFlavorError
		}
		fields[i], _ = strconv.Atoi(f.String())
	}

	return fields[0], fields[1], fields[2], nil
}

func (fl *FlavorsResp) GetIDByCondition(cpus, ram, disk int) string {
	name := fmt.Sprintf("g-c%dm%dd%d", cpus, ram, disk)
	for _, f := range fl.Flavors {
		if f.Name == name {
			return f.ID
		}
	}
	return ""
}

func (fl *FlavorsResp) GetIDByMemSize(memSize int) string {
	for _, f := range fl.Flavors {
		_, mem, _, err := getSpecFromFlavorName(f.Name)
		if err != nil {
			continue
		}
		if mem == memSize {
			return f.ID
		}
	}
	return ""
}

type IdentityReq struct {
	Auth struct {
		PasswordCredentials struct {
			UserName string `json:"username"`
			Password string `json:"password"`
		} `json:"passwordCredentials"`
		TenantID string `json:"tenantId"`
	} `json:"auth"`
}

type IdentityResp struct {
	Access struct {
		Token struct {
			Id      string `json:"id"`
			Expires string `json:"expires"`
		} `json:"token"`
	} `json:"access"`
}

func GetToken(cfg *config.Config) (string, string, error) {
	var auth IdentityReq
	auth.Auth.PasswordCredentials.UserName = cfg.Conoha.UserName
	auth.Auth.PasswordCredentials.Password = cfg.Conoha.Password
	auth.Auth.TenantID = cfg.Conoha.TenantID
	identData, _ := json.Marshal(auth)

	url, err := url.Parse(cfg.Conoha.Services.Identity)
	if err != nil {
		return "", "", err
	}
	url.Path = path.Join(url.Path, "tokens")

	req, err := http.NewRequest("POST", url.String(), bytes.NewBuffer(identData))
	if err != nil {
		return "", "", err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return "", "", errors.New(fmt.Sprintf("Authentication failed: %d", resp.StatusCode))
	}
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", "", err
	}
	var ident IdentityResp
	if err := json.Unmarshal(respBody, &ident); err != nil {
		return "", "", err
	}
	return ident.Access.Token.Id, ident.Access.Token.Expires, nil
}
