package gamesrv

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
	"time"

	"github.com/kofuk/premises/common/entity"
	"github.com/kofuk/premises/common/entity/runner"
	lm "github.com/kofuk/premises/common/mc/launchermeta"
	"github.com/kofuk/premises/runner/commands/levelinspect"
	"github.com/kofuk/premises/runner/exterior"
	"github.com/kofuk/premises/runner/fs"
	"github.com/kofuk/premises/runner/util"
)

func detectServerVersion() (string, error) {
	output := bytes.NewBuffer(nil)

	cmd := exec.Command("/opt/premises/bin/premises-runner", "--level-inspect")
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

	size := resp.ContentLength
	progress := make(chan int)
	reader := &util.ProgressReader{
		R:  resp.Body,
		Ch: progress,
	}

	go func() {
		ticker := time.NewTicker(time.Second)
		showNext := true
		var prevPercentage int64
		var totalUploaded int64
		for {
			select {
			case <-ticker.C:
				showNext = true
			case chunkSize, ok := <-progress:
				if !ok {
					return
				}

				totalUploaded += int64(chunkSize)

				if showNext && size > 0 {
					percentage := totalUploaded * 100 / size
					if percentage != prevPercentage {
						if err := exterior.SendMessage("serverStatus", runner.Event{
							Type: runner.EventStatus,
							Status: &runner.StatusExtra{
								EventCode: entity.EventGameDownload,
								Progress:  int(percentage),
							},
						}); err != nil {
							slog.Error("Unable to write server status", slog.Any("error", err))
						}
					}
					prevPercentage = percentage

					showNext = false
				}
			}
		}
	}()

	if _, err := io.Copy(outFile, reader); err != nil {
		return err
	}

	return nil
}

func DownloadServerJar(url, savePath string) error {
	if err := os.MkdirAll(fs.LocateDataFile("servers.d"), 0755); err != nil {
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
