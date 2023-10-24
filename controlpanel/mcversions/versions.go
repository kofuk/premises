package mcversions

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/kofuk/premises/controlpanel/entity"
)

const (
	versionManifestUrl = "https://launchermeta.mojang.com/mc/game/version_manifest.json"
)

var (
	ErrHttpFailure = errors.New("Failed to retrieve versions")
	ErrNotFound    = errors.New("Specified version not found")
)

type launcherMeta struct {
	Versions []struct {
		ID          string `json:"id"`
		Type        string `json:"type"`
		URL         string `json:"url"`
		ReleaseTime string `json:"releaseTime"`
	} `json:"versions"`
}

func fetchVersionManifest(ctx context.Context) (*launcherMeta, error) {
	req, err := http.NewRequest(http.MethodGet, versionManifestUrl, nil)
	if err != nil {
		return nil, err
	}
	req = req.WithContext(ctx)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, ErrHttpFailure
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, ErrHttpFailure
	}

	var meta launcherMeta
	if err := json.Unmarshal(data, &meta); err != nil {
		return nil, ErrHttpFailure
	}

	return &meta, nil
}

func GetVersions(ctx context.Context) ([]entity.MCVersion, error) {
	versionData, err := fetchVersionManifest(ctx)
	if err != nil {
		return nil, err
	}

	var result []entity.MCVersion

	for _, ver := range versionData.Versions {
		channel := ""
		if ver.Type == "release" {
			channel = "stable"
		} else if ver.Type == "snapshot" {
			channel = "snapshot"
		} else if ver.Type == "old_beta" {
			channel = "beta"
		} else if ver.Type == "old_alpha" {
			channel = "alpha"
		} else {
			channel = "unknown"
		}

		result = append(result, entity.MCVersion{
			Name:        ver.ID,
			IsStable:    ver.Type == "release",
			Channel:     channel,
			ReleaseDate: ver.ReleaseTime,
		})
	}

	return result, nil
}

type versionMeta struct {
	Downloads struct {
		Server struct {
			URL string `json:"url"`
		} `json:"server"`
	} `json:"downloads"`
}

func GetDownloadUrl(ctx context.Context, version string) (string, error) {
	// TODO: Use cached launcherMeta

	versionData, err := fetchVersionManifest(ctx)
	if err != nil {
		return "", err
	}

	versionMetaUrl := ""
	for _, ver := range versionData.Versions {
		if version == ver.ID {
			versionMetaUrl = ver.URL
		}
	}
	if versionMetaUrl == "" {
		return "", ErrNotFound
	}

	resp, err := http.Get(versionMetaUrl)
	if err != nil {
		return "", err
	}

	if resp.StatusCode != http.StatusOK {
		return "", ErrHttpFailure
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", ErrHttpFailure
	}

	var versinfo versionMeta
	if err := json.Unmarshal(data, &versinfo); err != nil {
		return "", ErrNotFound
	}

	url := versinfo.Downloads.Server.URL
	if url == "" {
		return "", ErrNotFound
	}

	return url, nil
}
