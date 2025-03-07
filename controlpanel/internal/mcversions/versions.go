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
	lm           *lm.LauncherMeta
	kvs          kvs.KeyValueStore
	overridenUrl string
}

func New(kvs kvs.KeyValueStore) MCVersionsService {
	var options []lm.Option

	manifestUrl := os.Getenv("PREMISES_MC_MANIFEST_URL")
	if manifestUrl != "" {
		options = append(options, lm.WithManifestURL(manifestUrl))
	}
	options = append(options, lm.WithHTTPClient(otelhttp.DefaultClient))

	service := MCVersionsService{
		lm:           lm.New(options...),
		kvs:          kvs,
		overridenUrl: manifestUrl,
	}

	return service
}

func (mcv MCVersionsService) GetVersions(ctx context.Context) ([]lm.VersionInfo, error) {
	{
		var result []lm.VersionInfo
		if err := mcv.kvs.Get(ctx, "mcversions:versions", &result); err != nil {
			slog.Error("Failed to get launchermeta from cache", slog.Any("error", err))
		} else {
			return result, nil
		}
	}

	versions, err := mcv.lm.GetVersionInfo(ctx)
	if err != nil {
		return nil, err
	}

	if err := mcv.kvs.Set(ctx, "mcversions:versions", versions, 24*time.Hour); err != nil {
		slog.Error("Failed to write version list cache", slog.Any("error", err))
	}

	return versions, nil
}

func (mcv MCVersionsService) GetServerInfo(ctx context.Context, version string) (*lm.ServerInfo, error) {
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

	serverInfo, err := mcv.lm.GetServerInfo(ctx, versionInfo)
	if err != nil {
		return nil, err
	}

	return serverInfo, nil
}

func (mcv MCVersionsService) GetOverridenManifestUrl() string {
	return mcv.overridenUrl
}
