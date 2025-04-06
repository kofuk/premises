package mcversions

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"time"

	"github.com/kofuk/premises/controlpanel/internal/kvs"
	lm "github.com/kofuk/premises/internal/mc/launchermeta"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

type MCVersionsService struct {
	lm           *lm.LauncherMetaClient
	kvs          kvs.KeyValueStore
	overridenURL string
}

func New(kvs kvs.KeyValueStore) *MCVersionsService {
	var options []lm.Option

	manifestUrl := os.Getenv("PREMISES_MC_MANIFEST_URL")
	if manifestUrl != "" {
		options = append(options, lm.WithManifestURL(manifestUrl))
	}
	options = append(options, lm.WithHTTPClient(otelhttp.DefaultClient))

	service := &MCVersionsService{
		lm:           lm.NewLauncherMetaClient(options...),
		kvs:          kvs,
		overridenURL: manifestUrl,
	}

	return service
}

func (mcv *MCVersionsService) GetVersions(ctx context.Context) ([]lm.VersionInfo, error) {
	{
		var result lm.VersionManifest
		if err := mcv.kvs.Get(ctx, "mcversions:versions", &result); err != nil {
			slog.Error("Failed to get launchermeta from cache", slog.Any("error", err))
		} else {
			return result.Versions, nil
		}
	}

	versions, err := mcv.lm.GetVersionInfo(ctx)
	if err != nil {
		return nil, err
	}

	if err := mcv.kvs.Set(ctx, "mcversions:versions", versions, 24*time.Hour); err != nil {
		slog.Error("Failed to write version list cache", slog.Any("error", err))
	}

	return versions.Versions, nil
}

func (mcv *MCVersionsService) GetLatestRelease(ctx context.Context) (string, error) {
	{
		var result lm.VersionManifest
		if err := mcv.kvs.Get(ctx, "mcversions:versions", &result); err != nil {
			slog.Error("Failed to get launchermeta from cache", slog.Any("error", err))
		} else {
			return result.Latest.Release, nil
		}
	}

	versions, err := mcv.lm.GetVersionInfo(ctx)
	if err != nil {
		return "", err
	}

	if err := mcv.kvs.Set(ctx, "mcversions:versions", versions, 24*time.Hour); err != nil {
		slog.Error("Failed to write version list cache", slog.Any("error", err))
	}

	return versions.Latest.Release, nil
}

type ServerInfo struct {
	DownloadURL   string
	LaunchCommand []string
	JavaVersion   int
}

func (mcv *MCVersionsService) GetServerInfo(ctx context.Context, version string) (*ServerInfo, error) {
	versions, err := mcv.GetVersions(ctx)
	if err != nil {
		return nil, err
	}

	var versionInfo lm.VersionInfo
	for _, ver := range versions {
		if version == ver.ID {
			versionInfo = ver
			break
		}
	}
	if versionInfo.ID == "" {
		return nil, errors.New("no matching version found")
	}

	versionMetaData, err := mcv.lm.GetVersionMetaData(ctx, versionInfo)
	if err != nil {
		return nil, err
	}

	return &ServerInfo{
		DownloadURL:   versionMetaData.Downloads.Server.URL,
		LaunchCommand: versionMetaData.Downloads.Server.CustomProperty.LaunchCommand,
		JavaVersion:   versionMetaData.JavaVersion.Major,
	}, nil
}

func (mcv *MCVersionsService) GetOverridenManifestURL() string {
	return mcv.overridenURL
}
