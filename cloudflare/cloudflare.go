package cloudflare

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"

	"github.com/kofuk/premises/config"
)

const (
	BaseURL = "https://api.cloudflare.com/client/v4"
)

type CloudflareResponse struct {
	Success bool `json:"success"`
	Errors  []struct {
		Message string `json:"message"`
	} `json:"errors"`
}

type ZonePropertiesResp struct {
	CloudflareResponse
	Result []struct {
		ID string `json:"id"`
	} `json:"result"`
}

func GetZoneID(cfg *config.Config) (string, error) {
	url, err := url.Parse(BaseURL)
	if err != nil {
		return "", err
	}
	url.Path = path.Join(url.Path, "zones")
	url.Query().Add("name", cfg.Cloudflare.DomainName)

	req, err := http.NewRequest("GET", url.String(), nil)
	if err != nil {
		return "", err
	}
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", cfg.Cloudflare.Token))

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", nil
	}
	defer resp.Body.Close()

	respData, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var result ZonePropertiesResp
	if err := json.Unmarshal(respData, &result); err != nil {
		return "", err
	}

	if !result.Success {
		var message string
		if len(result.Errors) > 0 {
			message = result.Errors[0].Message
		}
		return "", errors.New(fmt.Sprintf("Error retriving zone properties: errors: %v", message))
	}

	if len(result.Result) == 0 {
		return "", errors.New("Error retriving zone properties: no result returned")
	}

	return result.Result[0].ID, nil
}

type DNSRecordsResp struct {
	CloudflareResponse
	Result []struct {
		ID string `json:"id"`
	} `json:"result"`
}

func GetDNSRecordID(cfg *config.Config, zoneID, name, recordType string) (string, error) {
	url, err := url.Parse(BaseURL)
	if err != nil {
		return "", err
	}
	url.Path = path.Join(url.Path, "zones", zoneID, "dns_records")

	req, err := http.NewRequest("GET", url.String(), nil)
	if err != nil {
		return "", err
	}
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", cfg.Cloudflare.Token))
	query := req.URL.Query()
	query.Add("name", name)
	query.Add("type", recordType)
	query.Add("match", "all")
	req.URL.RawQuery = query.Encode()

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", nil
	}
	defer resp.Body.Close()

	respData, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var result DNSRecordsResp
	if err := json.Unmarshal(respData, &result); err != nil {
		return "", err
	}

	if !result.Success {
		var message string
		if len(result.Errors) > 0 {
			message = result.Errors[0].Message
		}
		return "", errors.New(fmt.Sprintf("Error retriving zone properties: errors: %v", message))
	}

	if len(result.Result) == 0 {
		return "", errors.New("Error retriving zone properties: no result returned")
	}

	return result.Result[0].ID, nil
}

func UpdateDNS(cfg *config.Config, zoneID, addr string, ipVer int) error {
	recordType := "A"
	if ipVer == 6 {
		recordType = "AAAA"
	}

	recordID, err := GetDNSRecordID(cfg, zoneID, cfg.Cloudflare.GameDomainName, recordType)
	if err != nil {
		return err
	}

	url, err := url.Parse(BaseURL)
	if err != nil {
		return err
	}
	url.Path = path.Join(url.Path, "zones", zoneID, "dns_records", recordID)

	req, err := http.NewRequest("PATCH", url.String(), bytes.NewBuffer([]byte(fmt.Sprintf("{\"content\":\"%s\"}", addr))))
	if err != nil {
		return err
	}
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", cfg.Cloudflare.Token))

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil
	}
	defer resp.Body.Close()

	respData, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	var result CloudflareResponse
	if err := json.Unmarshal(respData, &result); err != nil {
		return err
	}

	if !result.Success {
		var message string
		if len(result.Errors) > 0 {
			message = result.Errors[0].Message
		}
		return errors.New(fmt.Sprintf("Error retriving zone properties: errors: %v", message))
	}

	return nil
}
