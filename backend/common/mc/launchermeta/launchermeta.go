package launchermeta

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

const (
	mojangManifest = "https://launchermeta.mojang.com/mc/game/version_manifest.json"
)

type LauncherMetaClient struct {
	httpClient  *http.Client
	manifestURL string
}

type Option func(p *LauncherMetaClient)

func WithManifestURL(url string) Option {
	return func(p *LauncherMetaClient) {
		p.manifestURL = url
	}
}

func WithHTTPClient(client *http.Client) Option {
	return func(p *LauncherMetaClient) {
		p.httpClient = client
	}
}

type VersionInfo struct {
	ID          string `json:"id"`
	Type        string `json:"type"`
	URL         string `json:"url"`
	ReleaseTime string `json:"releaseTime"`
}

type VersionManifest struct {
	Latest struct {
		Release  string `json:"release"`
		Snapshot string `json:"snapshot"`
	}
	Versions []VersionInfo `json:"versions"`
}

func NewLauncherMetaClient(options ...Option) *LauncherMetaClient {
	provider := &LauncherMetaClient{
		httpClient:  http.DefaultClient,
		manifestURL: mojangManifest,
	}

	for _, opt := range options {
		opt(provider)
	}

	return provider
}

func (lm *LauncherMetaClient) GetVersionInfo(ctx context.Context) (*VersionManifest, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, lm.manifestURL, nil)
	if err != nil {
		return nil, err
	}

	resp, err := lm.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		io.CopyN(io.Discard, resp.Body, 1024)
		return nil, fmt.Errorf("failed to retrieve launchermeta: %s", resp.Status)
	}

	var manifest VersionManifest
	if err := json.NewDecoder(resp.Body).Decode(&manifest); err != nil {
		return nil, err
	}

	return &manifest, nil
}

type VersionMetaData struct {
	Downloads struct {
		Server struct {
			URL            string `json:"url"`
			CustomProperty struct {
				LaunchCommand []string `json:"launchCommand"`
			} `json:"x-premises"`
		} `json:"server"`
	} `json:"downloads"`
	JavaVersion struct {
		Major int `json:"majorVersion"`
	} `json:"javaVersion"`
}

func (lm *LauncherMetaClient) GetVersionMetaData(ctx context.Context, version VersionInfo) (*VersionMetaData, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, version.URL, nil)
	if err != nil {
		return nil, err
	}

	resp, err := lm.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get version metadata")
	}

	var versionMeta VersionMetaData
	if err := json.NewDecoder(resp.Body).Decode(&versionMeta); err != nil {
		return nil, err
	}

	return &versionMeta, nil
}
