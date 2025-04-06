package game

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"strings"

	"github.com/kofuk/premises/internal/entity"
	"github.com/kofuk/premises/internal/entity/runner"
	lm "github.com/kofuk/premises/internal/mc/launchermeta"
	"github.com/kofuk/premises/runner/internal/env"
	"github.com/kofuk/premises/runner/internal/system"
	"github.com/kofuk/premises/runner/internal/util"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

func DetectAndUpdateVersion(ctx context.Context, config *runner.Config) error {
	version, err := DetectServerVersion()
	if err != nil {
		return err
	}
	slog.Debug("Server version detected", slog.String("version", version))

	var options []lm.Option

	if config.GameConfig.Server.ManifestOverride != "" {
		options = append(options, lm.WithManifestURL(config.GameConfig.Server.ManifestOverride))
	}
	options = append(options, lm.WithHTTPClient(otelhttp.DefaultClient))

	fetcher := lm.NewLauncherMetaClient(options...)
	versions, err := fetcher.GetVersionInfo(ctx)
	if err != nil {
		return err
	}

	var versionInfo lm.VersionInfo
	for _, ver := range versions.Versions {
		if ver.ID == version {
			versionInfo = ver
			break
		}
	}
	if versionInfo.ID == "" {
		return errors.New("no matching version found")
	}

	versionMetaData, err := fetcher.GetVersionMetaData(ctx, versionInfo)
	if err != nil {
		return err
	}

	if versionMetaData.Downloads.Server.URL != "" {
		config.GameConfig.Server.DownloadUrl = versionMetaData.Downloads.Server.URL
		config.GameConfig.Server.Version = version
		config.GameConfig.Server.JavaVersion = versionMetaData.JavaVersion.Major

		return nil
	}

	return errors.New("version found, but download URL was not found")
}

func isExecutableFile(path string) bool {
	file, err := os.Open(path)
	if err != nil {
		slog.Error("Unable to open file", slog.Any("error", err))
		return false
	}
	defer file.Close()

	buf := make([]byte, 4)
	if _, err := file.Read(buf); err != nil {
		slog.Error("Unable to read file", slog.Any("error", err))
		return false
	}

	// ELF or shell script
	return (buf[0] == 0x7F && buf[1] == 'E' && buf[2] == 'L' && buf[3] == 'F') || (buf[0] == '#' && buf[1] == '!')
}

func downloadServerJar(ctx context.Context, url, savePath string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}

	resp, err := otelhttp.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		io.CopyN(io.Discard, resp.Body, 10*1024)
		return fmt.Errorf("download failed with status: %d", resp.StatusCode)
	}

	outFile, err := os.Create(savePath)
	if err != nil {
		io.Copy(io.Discard, resp.Body)
		return err
	}
	defer outFile.Close()

	reader := util.NewProgressReader(ctx, resp.Body, entity.EventGameDownload, int(resp.ContentLength))

	if _, err := io.Copy(outFile, reader); err != nil {
		return err
	}

	return nil
}

func DownloadServerJar(ctx context.Context, url, savePath string) error {
	if err := os.MkdirAll(env.DataPath("servers.d"), 0755); err != nil {
		return err
	}

	tmpPath := savePath + ".download"
	if err := downloadServerJar(ctx, url, tmpPath); err != nil {
		return err
	}

	if isExecutableFile(tmpPath) {
		if err := os.Chmod(tmpPath, 0755); err != nil {
			return err
		}
	}

	if err := os.Rename(tmpPath, savePath); err != nil {
		return err
	}

	return nil
}

func getJavaPathFromInstalledVersion(ctx context.Context, version int) (string, error) {
	output, err := system.CmdOutput(ctx, "update-alternatives", []string{"--list", "java"})
	if err != nil {
		return "", err
	}

	candidates := strings.Split(strings.TrimRight(output, "\r\n"), "\n")
	slog.Debug("Installed java versions", slog.Any("versions", candidates))

	for _, path := range candidates {
		if strings.Contains(path, fmt.Sprintf("-%d-", version)) {
			return path, nil
		}
	}

	return "", errors.New("not found")
}

func findJavaPath(ctx context.Context, version int) string {
	if version == 0 {
		slog.Info("Version not specified. Using the system default")
		return "java"
	}

	path, err := getJavaPathFromInstalledVersion(ctx, version)
	if err != nil {
		slog.Warn("Error finding java installation. Using the system default", slog.Any("error", err))
		return "java"
	}

	slog.Info("Found java installation matching requested version", slog.String("path", path), slog.Int("requested_version", version))

	return path
}
