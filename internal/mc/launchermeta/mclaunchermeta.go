package mclaunchermeta

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

const (
	mojangManifest = "https://launchermeta.mojang.com/mc/game/version_manifest.json"
)

type LauncherMeta struct {
	httpClient  *http.Client
	manifestURL string
}

type Option func(p *LauncherMeta)

func WithManifestURL(url string) Option {
	return func(p *LauncherMeta) {
		p.manifestURL = url
	}
}

func WithHTTPClient(client *http.Client) Option {
	return func(p *LauncherMeta) {
		p.httpClient = client
	}
}

type VersionInfo struct {
	ID          string `json:"id"`
	Type        string `json:"type"`
	URL         string `json:"url"`
	ReleaseTime string `json:"releaseTime"`
}

type launcherMetaData struct {
	Versions []VersionInfo `json:"versions"`
}

func New(options ...Option) *LauncherMeta {
	provider := &LauncherMeta{
		httpClient:  http.DefaultClient,
		manifestURL: mojangManifest,
	}

	for _, opt := range options {
		opt(provider)
	}

	return provider
}

func (lm *LauncherMeta) GetVersionInfo(ctx context.Context) ([]VersionInfo, error) {
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
		return nil, fmt.Errorf("failed to retrieve launchermeta")
	}

	var meta launcherMetaData
	if err := json.NewDecoder(resp.Body).Decode(&meta); err != nil {
		return nil, err
	}

	return meta.Versions, nil
}

type versionMetaResp struct {
	Downloads struct {
		Server struct {
			URL            string `json:"url"`
			CustomProperty struct {
				LaunchCommand []string `json:"launchCommand"`
			} `json:"x-premises"`
		} `json:"server"`
	} `json:"downloads"`
	JavaVersion struct {
		Marjor int `json:"majorVersion"`
	} `json:"javaVersion"`
}

type ServerInfo struct {
	DownloadURL   string
	LaunchCommand []string
	JavaVersion   int
}

func (lm *LauncherMeta) GetServerInfo(ctx context.Context, version VersionInfo) (*ServerInfo, error) {
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

	var versionMeta versionMetaResp
	if err := json.NewDecoder(resp.Body).Decode(&versionMeta); err != nil {
		return nil, err
	}

	result := &ServerInfo{
		DownloadURL:   versionMeta.Downloads.Server.URL,
		LaunchCommand: versionMeta.Downloads.Server.CustomProperty.LaunchCommand,
		JavaVersion:   versionMeta.JavaVersion.Marjor,
	}

	return result, nil
}
