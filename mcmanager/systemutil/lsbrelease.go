package systemutil

import (
	"bufio"
	"errors"
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

func readDistroFromLsbRelease(file io.Reader) (string, error) {
	reader := bufio.NewReader(file)
	for {
		line, _, err := reader.ReadLine()
		if err != nil {
			if err != io.EOF {
				return "", err
			}
			break
		}
		lineStr := string(line)
		fieldSepPos := strings.IndexRune(lineStr, '=')

		if fieldSepPos < 0 {
			continue
		}

		fieldName := lineStr[:fieldSepPos]
		fieldValue := lineStr[fieldSepPos+1:]
		if fieldName == "DISTRIB_DESCRIPTION" {
			if len(fieldValue) >= 2 && fieldValue[0] == '"' && fieldValue[len(fieldValue)-1] == '"' {
				return fieldValue[1 : len(fieldValue)-1], nil
			}
			return fieldValue, nil
		}
	}

	return "", errors.New("DISTRIB_DESCRIPTION not found")
}

func GetHostOS() (string, error) {
	file, err := os.Open("/etc/lsb-release")
	if err != nil {
		return "", err
	}
	defer file.Close()

	return readDistroFromLsbRelease(file)
}

func GetSystemVersion() *SystemInfo {
	hostOS, err := GetHostOS()
	if err != nil {
		log.WithError(err).Error("Error retrieving host OS")
		hostOS = "unknown"
	}

	return &SystemInfo{
		PremisesVersion: metadata.Revision,
		HostOS:          hostOS,
	}
}
