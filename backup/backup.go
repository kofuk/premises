package backup

import (
	"errors"
	"regexp"
	"strconv"
	"strings"

	"chronoscoper.com/premises/config"
	"github.com/t3rm1n4l/go-mega"
)

type WorldBackup struct {
	ServerName string `json:"serverName"`
	WorldName  string `json:"worldName"`
	Generation int    `json:"generation"`
}

func getFolderRef(m *mega.Mega, parent *mega.Node, name string) (*mega.Node, error) {
	children, err := m.FS.GetChildren(parent)
	if err != nil {
		return nil, err
	}
	for _, folder := range children {
		if folder.GetName() == name && folder.GetType() == mega.FOLDER {
			return folder, nil
		}
	}
	return nil, errors.New("No such folder")
}

func getCloudWorldsFolder(m *mega.Mega) (*mega.Node, error) {
	dataRoot, err := getFolderRef(m, m.FS.GetRoot(), "premises")
	if err != nil {
		return nil, err
	}
	worldsFolder, err := getFolderRef(m, dataRoot, "worlds")
	if err != nil {
		return nil, err
	}
	return worldsFolder, nil
}

var archiveExtensionRegexp = regexp.MustCompile("\\.tar\\.xz$")

func GetBackupList(cfg *config.Config) ([]WorldBackup, error) {
	if cfg.Mega.Email == "" {
		return nil, errors.New("Mega credential is not set")
	}

	m := mega.New()
	if err := m.Login(cfg.Mega.Email, cfg.Mega.Password); err != nil {
		return nil, err
	}

	worldsFolder, err := getCloudWorldsFolder(m)
	if err != nil {
		return nil, err
	}

	backups, err := m.FS.GetChildren(worldsFolder)
	if err != nil {
		return nil, err
	}

	var result []WorldBackup
	for _, backup := range backups {
		name := archiveExtensionRegexp.ReplaceAllString(backup.GetName(), "")
		components := strings.Split(name, "@")
		if len(components) != 3 {
			continue
		}

		generation, err := strconv.Atoi(components[2])
		if err != nil {
			if components[2] == "latest" {
				generation = 0
			} else {
				continue
			}
		}

		result = append(result, WorldBackup{
			ServerName: components[0],
			WorldName:  components[1],
			Generation: generation,
		})
	}

	return result, nil
}
