package serverjar

import (
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/kofuk/premises/internal/entity"
	"github.com/kofuk/premises/internal/mc/launchermeta"
	"github.com/kofuk/premises/internal/retry"
	"github.com/kofuk/premises/runner/internal/commands/mclauncher/core"
	"github.com/kofuk/premises/runner/internal/util"
)

const StateKeyMinecraftVersion = "github.com/kofuk/premises/runner/mclauncher/middleware/serverjar.MinecraftVersion"

type ServerJarMiddleware struct {
	launcherMetaClient *launchermeta.LauncherMetaClient
	httpClient         *http.Client
}

var _ core.Middleware = (*ServerJarMiddleware)(nil)

func NewServerJarMiddleware(launcherMetaClient *launchermeta.LauncherMetaClient, httpClient *http.Client) *ServerJarMiddleware {
	return &ServerJarMiddleware{
		launcherMetaClient: launcherMetaClient,
		httpClient:         httpClient,
	}
}

func (m *ServerJarMiddleware) downloadMatchingVersion(c core.LauncherContext, desiredVersion string, destination string) error {
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

		resp, err := m.httpClient.Do(req)
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
		// Best effort to clean up the incomplete download
		os.Remove(destination)
		return err
	}

	return nil
}

func (m *ServerJarMiddleware) downloadIfNotExists(c core.LauncherContext) error {
	version := c.Settings().GetMinecraftVersion()
	serverPath := c.Env().GetDataPath("servers.d", version+".jar")

	if stat, err := os.Stat(serverPath); err == nil {
		if stat.Mode().IsRegular() && stat.Size() > 0 {
			return nil
		}

		// If the file exists but is not a regular file or is empty, remove it and try to download again.
		if err := os.RemoveAll(serverPath); err != nil {
			return err
		}
	} else if !os.IsNotExist(err) {
		return err
	}

	if err := m.downloadMatchingVersion(c, version, serverPath); err != nil {
		return err
	}

	return nil
}

func (m *ServerJarMiddleware) cleanupDataDir(c core.LauncherContext) error {
	gameDataDir := c.Env().GetDataPath("gamedata")
	ents, err := os.ReadDir(gameDataDir)
	if err != nil {
		return err
	}

	var errs []error
	for _, ent := range ents {
		if ent.Name() == "server.properties" || ent.Name() == "world" || strings.HasPrefix(ent.Name(), "ss@") {
			continue
		}
		if err := os.RemoveAll(filepath.Join(gameDataDir, ent.Name())); err != nil {
			errs = append(errs, err)
		}
	}

	return errors.Join(errs...)
}

func (m *ServerJarMiddleware) Wrap(next core.HandlerFunc) core.HandlerFunc {
	return func(c core.LauncherContext) error {
		if err := m.downloadIfNotExists(c); err != nil {
			return err
		}
		c.Settings().SetServerPath(c.Env().GetDataPath("servers.d", c.Settings().GetMinecraftVersion()+".jar"))

		oldVersion, err := c.State().GetState(c.Context(), StateKeyMinecraftVersion)
		if err != nil || oldVersion != c.Settings().GetMinecraftVersion() {
			// Minecraft version has changed, clean up the old data directory.
			if err := m.cleanupDataDir(c); err != nil {
				return err
			}
		}
		c.State().SetState(c.Context(), StateKeyMinecraftVersion, c.Settings().GetMinecraftVersion())

		return next(c)
	}
}
