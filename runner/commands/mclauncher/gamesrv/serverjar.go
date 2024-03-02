package gamesrv

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os/exec"

	"github.com/kofuk/premises/common/entity/runner"
	lm "github.com/kofuk/premises/common/mc/launchermeta"
	"github.com/kofuk/premises/runner/commands/levelinspect"
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

	fetcher := lm.New()
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

	if serverInfo.URL != "" {
		config.Server.DownloadUrl = serverInfo.URL
		config.Server.Version = version

		return nil
	}

	return errors.New("Version found, but download URL was not found")
}
