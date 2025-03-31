package serverjar

import (
	"errors"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/kofuk/premises/internal/entity"
	"github.com/kofuk/premises/internal/mc/launchermeta"
	"github.com/kofuk/premises/internal/retry"
	"github.com/kofuk/premises/runner/internal/commands/mclauncher/core"
	"github.com/kofuk/premises/runner/internal/env"
	"github.com/kofuk/premises/runner/internal/util"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

type ServerJarMiddleware struct {
	pathProvider       env.PathProvider
	launcherMetaClient *launchermeta.LauncherMetaClient
}

var _ core.Middleware = (*ServerJarMiddleware)(nil)

func NewServerJarMiddleware(pathProvider env.PathProvider, launcherMetaClient *launchermeta.LauncherMetaClient) *ServerJarMiddleware {
	return &ServerJarMiddleware{
		pathProvider:       pathProvider,
		launcherMetaClient: launcherMetaClient,
	}
}

func (m *ServerJarMiddleware) downloadMatchingVersion(c *core.LauncherContext, desiredVersion string, destination string) error {
	versions, err := retry.Retry(func() (*launchermeta.VersionManifest, error) {
		return m.launcherMetaClient.GetVersionInfo(c.Context())
	}, time.Minute)
	if err != nil {
		return err
	}

	var matchedVersion *launchermeta.VersionInfo
	for _, version := range versions.Versions {
		if version.ID == desiredVersion {
			matchedVersion = &version
			break
		}
	}

	if matchedVersion == nil {
		return errors.New("version not found")
	}

	versionMetaData, err := retry.Retry(func() (*launchermeta.VersionMetaData, error) {
		return m.launcherMetaClient.GetVersionMetaData(c.Context(), *matchedVersion)
	}, time.Minute)
	if err != nil {
		return err
	}

	outFile, err := os.Create(destination)
	if err != nil {
		return err
	}
	defer outFile.Close()

	_, err = retry.Retry(func() (_ retry.Void, err error) {
		defer func() {
			if err != nil {
				outFile.Truncate(0)
				outFile.Seek(0, io.SeekStart)
			}
		}()

		var req *http.Request
		req, err = http.NewRequestWithContext(c.Context(), http.MethodGet, versionMetaData.Downloads.Server.URL, nil)
		if err != nil {
			return retry.V, err
		}

		resp, err := otelhttp.DefaultClient.Do(req)
		if err != nil {
			return retry.V, err
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			io.CopyN(io.Discard, resp.Body, 10*1024)
			err = errors.New("failed to download server jar")
			return
		}

		_, err = io.Copy(outFile, util.NewProgressReader(c.Context(), resp.Body, entity.EventGameDownload, int(resp.ContentLength)))
		if err != nil {
			return
		}

		return
	}, 5*time.Minute)
	if err != nil {
		return err
	}

	return nil
}

func (m *ServerJarMiddleware) downloadIfNotExists(c *core.LauncherContext) error {
	version := c.Settings().GetMinecraftVersion()
	serverPath := m.pathProvider.GetDataPath("servers.d", version+".jar")

	if _, err := os.Stat(serverPath); err == nil {
		return nil
	} else if !os.IsNotExist(err) {
		return err
	}

	if err := m.downloadMatchingVersion(c, version, serverPath); err != nil {
		return err
	}

	return nil
}

func (m *ServerJarMiddleware) Wrap(next core.HandlerFunc) core.HandlerFunc {
	return func(c *core.LauncherContext) error {
		if err := m.downloadIfNotExists(c); err != nil {
			return err
		}
		c.Settings().SetServerPath(m.pathProvider.GetDataPath("servers.d", c.Settings().GetMinecraftVersion()+".jar"))

		return next(c)
	}
}
