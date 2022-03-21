package statusapi

import (
	"bufio"
	"io"
	"os"
	"strings"

	"github.com/kofuk/premises/mcmanager/metadata"
	log "github.com/sirupsen/logrus"
)

type SystemInfo struct {
	PremisesVersion string `json:"premisesVersion"`
	HostOS          string `json:"hostOS"`
}

func GetHostOS() string {
	file, err := os.Open("/etc/lsb-release")
	if err != nil {
		log.WithError(err).Error("Failed to open lsb_release")
		return "unknown"
	}
	defer file.Close()

	reader := bufio.NewReader(file)
	for {
		line, _, err := reader.ReadLine()
		if err != nil {
			if err != io.EOF {
				log.WithError(err).Error("Error reading lsb_release")
			}
			break
		}
		fields := strings.Split(string(line), "=")
		if len(fields) != 2 {
			continue
		}
		if fields[0] == "DISTRIB_DESCRIPTION" {
			return strings.ReplaceAll(fields[1], "\"", "")
		}
	}
	return "unknown"
}

func GetSystemInfo() *SystemInfo {
	return &SystemInfo{
		PremisesVersion: metadata.Revision,
		HostOS: GetHostOS(),
	}
}
