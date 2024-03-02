package mclaunchermeta

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

type LauncherMeta struct {
	manifestURL string
}

type Option func(p *LauncherMeta)

func ManifestURL(url string) Option {
	return func(p *LauncherMeta) {
		p.manifestURL = url
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

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Failed to retrieve launchermeta")
	}

	var meta launcherMetaData
	if err := json.Unmarshal(data, &meta); err != nil {
		return nil, err
	}

	return meta.Versions, nil
}

type ServerInfo struct {
	URL string `json:"url"`
}

type versionMetaData struct {
	Downloads struct {
		Server ServerInfo `json:"server"`
	} `json:"downloads"`
}

func (lm *LauncherMeta) GetServerInfo(ctx context.Context, version VersionInfo) (*ServerInfo, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, version.URL, nil)
	if err != nil {
		return nil, err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Failed to get version metadata")
	}

	var versinfo versionMetaData
	if err := json.Unmarshal(data, &versinfo); err != nil {
		return nil, err
	}

	return &versinfo.Downloads.Server, nil
}
