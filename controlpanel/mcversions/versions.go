package mcversions

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	lm "github.com/kofuk/premises/common/mc/launchermeta"
	"github.com/kofuk/premises/controlpanel/kvs"
)

const (
	mojangManifest = "https://launchermeta.mojang.com/mc/game/version_manifest.json"
)

type MCVersionsService struct {
	lm  *lm.LauncherMeta
	kvs kvs.KeyValueStore
}

func New(kvs kvs.KeyValueStore) MCVersionsService {
	provider := MCVersionsService{
		lm:  lm.New(),
		kvs: kvs,
	}

	return provider
}

func (self MCVersionsService) GetVersions(ctx context.Context) ([]lm.VersionInfo, error) {
	{
		var result []lm.VersionInfo
		if err := self.kvs.Get(ctx, "mcversions:versions", &result); err != nil {
			slog.Error("Failed to get launchermeta from cache", slog.Any("error", err))
		} else {
			return result, nil
		}
	}

	versions, err := self.lm.GetVersionInfo(ctx)
	if err != nil {
		return nil, err
	}

	if err := self.kvs.Set(ctx, "mcversions:versions", versions, 24*time.Hour); err != nil {
		slog.Error("Failed to write version list cache", slog.Any("error", err))
	}

	return versions, nil
}

func (self MCVersionsService) GetDownloadURL(ctx context.Context, version string) (string, error) {
	{
		var result string
		if err := self.kvs.Get(ctx, fmt.Sprintf("mcversions:v%s", version), &result); err != nil {
			slog.Error("Failed to get version data from cache", slog.Any("error", err), slog.String("version", version))
		} else {
			return result, nil
		}
	}

	versions, err := self.GetVersions(ctx)
	if err != nil {
		return "", err
	}

	var versionInfo lm.VersionInfo
	for _, ver := range versions {
		if version == ver.ID {
			versionInfo = ver
			break
		}
	}
	if versionInfo.ID == "" {
		return "", errors.New("No matching version found")
	}

	serverInfo, err := self.lm.GetServerInfo(ctx, versionInfo)
	if err != nil {
		return "", err
	}

	if err := self.kvs.Set(ctx, fmt.Sprintf("mcversions:v%s", version), serverInfo.URL, -1); err != nil {
		slog.Error("Failed to write mcversions cache", slog.Any("error", err), slog.String("version", version))
	}

	return serverInfo.URL, nil
}
