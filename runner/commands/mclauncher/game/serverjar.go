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
	"github.com/kofuk/premises/runner/env"
	"github.com/kofuk/premises/runner/system"
	"github.com/kofuk/premises/runner/util"
)

func DetectAndUpdateVersion(config *runner.Config) error {
	version, err := DetectServerVersion()
	if err != nil {
		return err
	}
	slog.Debug("Server version detected", slog.String("version", version))

	var options []lm.Option

	if config.GameConfig.Server.ManifestOverride != "" {
		options = append(options, lm.ManifestURL(config.GameConfig.Server.ManifestOverride))
	}

	fetcher := lm.New(options...)
	versions, err := fetcher.GetVersionInfo(context.TODO())
	if err != nil {
		return err
	}

	var versionInfo lm.VersionInfo
	for _, ver := range versions {
		if ver.ID == version {
			versionInfo = ver
			break
		}
	}
	if versionInfo.ID == "" {
		return errors.New("no matching version found")
	}

	serverInfo, err := fetcher.GetServerInfo(context.TODO(), versionInfo)
	if err != nil {
		return err
	}

	if serverInfo.DownloadURL != "" {
		config.GameConfig.Server.DownloadUrl = serverInfo.DownloadURL
		config.GameConfig.Server.Version = version
		config.GameConfig.Server.JavaVersion = serverInfo.JavaVersion

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

func downloadServerJar(url, savePath string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK {
		io.Copy(io.Discard, resp.Body)
		return fmt.Errorf("download failed with status: %d", resp.StatusCode)
	}

	outFile, err := os.Create(savePath)
	if err != nil {
		io.Copy(io.Discard, resp.Body)
		return err
	}
	defer outFile.Close()

	reader := util.NewProgressReader(resp.Body, entity.EventGameDownload, int(resp.ContentLength))

	if _, err := io.Copy(outFile, reader); err != nil {
		return err
	}

	return nil
}

func DownloadServerJar(url, savePath string) error {
	if err := os.MkdirAll(env.DataPath("servers.d"), 0755); err != nil {
		return err
	}

	tmpPath := savePath + ".download"
	if err := downloadServerJar(url, tmpPath); err != nil {
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

func getJavaPathFromInstalledVersion(version int) (string, error) {
	output, err := system.CmdOutput("update-alternatives", []string{"--list", "java"})
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

func findJavaPath(version int) string {
	if version == 0 {
		slog.Info("Version not specified. Using the system default")
		return "java"
	}

	path, err := getJavaPathFromInstalledVersion(version)
	if err != nil {
		slog.Warn("Error finding java installation. Using the system default", slog.Any("error", err))
		return "java"
	}

	slog.Info("Found java installation matching requested version", slog.String("path", path), slog.Int("requested_version", version))

	return path
}
