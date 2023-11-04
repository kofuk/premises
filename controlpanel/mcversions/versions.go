package mcversions

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/kofuk/premises/controlpanel/caching"
	log "github.com/sirupsen/logrus"
)

const (
	mojangManifest = "https://launchermeta.mojang.com/mc/game/version_manifest.json"
)

type MCVersionProvider struct {
	cacher      caching.Cacher
	manifestURL string
}

type Option func(p *MCVersionProvider)

func ManifestURL(url string) Option {
	return func(p *MCVersionProvider) {
		p.manifestURL = url
	}
}

func New(cacher caching.Cacher, options ...Option) MCVersionProvider {
	provider := MCVersionProvider{
		cacher:      cacher,
		manifestURL: mojangManifest,
	}

	for _, opt := range options {
		opt(&provider)
	}

	return provider
}

func (self MCVersionProvider) fetchVersionManifest(ctx context.Context) (*launcherMeta, error) {
	req, err := http.NewRequest(http.MethodGet, self.manifestURL, nil)
	if err != nil {
		return nil, err
	}
	req = req.WithContext(ctx)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Failed to retrieve launchermeta")
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var meta launcherMeta
	if err := json.Unmarshal(data, &meta); err != nil {
		return nil, err
	}

	return &meta, nil
}

type VersionInfo struct {
	ID          string `json:"id"`
	Type        string `json:"type"`
	URL         string `json:"url"`
	ReleaseTime string `json:"releaseTime"`
}

type launcherMeta struct {
	Versions []VersionInfo `json:"versions"`
}

func (self MCVersionProvider) GetVersions(ctx context.Context) ([]VersionInfo, error) {
	{
		var result launcherMeta
		if err := self.cacher.Get(ctx, "mcversions:launchermeta", &result); err != nil {
			log.WithError(err).Error("Failed to get launchermeta from cache")
		} else {
			return result.Versions, nil
		}
	}

	launcherMeta, err := self.fetchVersionManifest(ctx)
	if err != nil {
		return nil, err
	}

	if err := self.cacher.Set(ctx, "mcversions:launchermeta", launcherMeta, 24*time.Hour); err != nil {
		log.WithError(err).Error("Failed to write version list cache")
	}

	return launcherMeta.Versions, nil
}

type versionMeta struct {
	Downloads struct {
		Server struct {
			URL string `json:"url"`
		} `json:"server"`
	} `json:"downloads"`
}

func (self MCVersionProvider) fetchDownloadURL(ctx context.Context, versionMetaURL string) (string, error) {
	req, err := http.NewRequest(http.MethodGet, versionMetaURL, nil)
	if err != nil {
		return "", err
	}
	req = req.WithContext(ctx)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("Failed to get version metadata")
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var versinfo versionMeta
	if err := json.Unmarshal(data, &versinfo); err != nil {
		return "", err
	}

	url := versinfo.Downloads.Server.URL
	if url == "" {
		return "", fmt.Errorf("Download URL for version is not set")
	}

	return url, nil
}

func (self MCVersionProvider) GetDownloadURL(ctx context.Context, version string) (string, error) {
	{
		var result string
		if err := self.cacher.Get(ctx, fmt.Sprintf("mcversions:v%s", version), &result); err != nil {
			log.WithError(err).WithField("version", version).Error("Failed to get version data from cache")
		} else {
			return result, nil
		}
	}

	versionData, err := self.GetVersions(ctx)
	if err != nil {
		return "", err
	}

	versionMetaURL := ""
	for _, ver := range versionData {
		if version == ver.ID {
			versionMetaURL = ver.URL
		}
	}
	if versionMetaURL == "" {
		return "", fmt.Errorf("Specified version not found")
	}

	url, err := self.fetchDownloadURL(ctx, versionMetaURL)
	if err != nil {
		return "", err
	}

	if err := self.cacher.Set(ctx, fmt.Sprintf("mcversions:v%s", version), url, -1); err != nil {
		log.WithError(err).WithField("version", version).Error("Failed to write mcversions cache")
	}

	return url, nil
}
