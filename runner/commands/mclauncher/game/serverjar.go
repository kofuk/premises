package game

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/exec"
	"strings"

	"github.com/kofuk/premises/common/entity"
	"github.com/kofuk/premises/common/entity/runner"
	lm "github.com/kofuk/premises/common/mc/launchermeta"
	"github.com/kofuk/premises/runner/commands/levelinspect"
	"github.com/kofuk/premises/runner/fs"
	"github.com/kofuk/premises/runner/systemutil"
	"github.com/kofuk/premises/runner/util"
)

func detectServerVersion() (string, error) {
	output := bytes.NewBuffer(nil)

	cmd := exec.Command(fs.DataPath("bin/premises-runner"), "--level-inspect")
	cmd.Stdout = output
	if err := cmd.Run(); err != nil {
		return "", err
	}

	var result levelinspect.Result
	if err := json.Unmarshal(output.Bytes(), &result); err != nil {
		return "", err
	}

	return result.ServerVersion, nil
}

func DetectAndUpdateVersion(config *runner.Config) error {
	version, err := detectServerVersion()
	if err != nil {
		return err
	}
	slog.Debug("Server version detected", slog.String("version", version))

	var options []lm.Option

	if config.Server.ManifestOverride != "" {
		options = append(options, lm.ManifestURL(config.Server.ManifestOverride))
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
		return errors.New("No matching version found")
	}

	serverInfo, err := fetcher.GetServerInfo(context.TODO(), versionInfo)
	if err != nil {
		return err
	}

	if serverInfo.DownloadURL != "" {
		config.Server.DownloadUrl = serverInfo.DownloadURL
		config.Server.Version = version
		config.Server.JavaVersion = serverInfo.JavaVersion

		return nil
	}

	return errors.New("Version found, but download URL was not found")
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
		return errors.New(fmt.Sprintf("Download failed with status: %d", resp.StatusCode))
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
	if err := os.MkdirAll(fs.DataPath("servers.d"), 0755); err != nil {
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
	output, err := systemutil.CmdOutput("update-alternatives", []string{"--list", "java"})
	if err != nil {
		return "", err
	}

	candidates := strings.Split(strings.TrimRight(output, "\r\n"), "\n")
	slog.Debug("Installed java versions", slog.Any("versions", candidates))

	for _, path := range candidates {
		if strings.Index(path, fmt.Sprintf("-%d-", version)) >= 0 {
			return path, nil
		}
	}

	return "", errors.New("Not found")
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