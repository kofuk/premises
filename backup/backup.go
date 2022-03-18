package backup

import (
	"errors"
	"regexp"
	"sort"
	"strconv"

	"github.com/kofuk/go-mega"
	"github.com/kofuk/premises/config"
	log "github.com/sirupsen/logrus"
)

type GenerationInfo struct {
	Gen       string `json:"gen"`
	ID        string `json:"id"`
	Timestamp int    `json:"timestamp"`
}

type WorldBackup struct {
	WorldName   string           `json:"worldName"`
	Generations []GenerationInfo `json:"generations"`
}

func getFolderRef(m *mega.Mega, parent *mega.Node, name string) (*mega.Node, error) {
	children, err := m.FS.GetChildren(parent)
	if err != nil {
		return nil, err
	}
	for _, folder := range children {
		if folder.GetName() == name && folder.GetType() == mega.TypeFolder {
			return folder, nil
		}
	}
	return nil, errors.New("No such folder")
}

func getCloudWorldsFolder(cfg *config.Config, m *mega.Mega) (*mega.Node, error) {
	dataRoot, err := getFolderRef(m, m.FS.GetRoot(), "premises")
	if err != nil {
		return nil, err
	}

	var worldFolderName string
	if cfg.Debug.Runner {
		worldFolderName = "worlds.dev"
	} else {
		worldFolderName = "worlds"
	}

	worldsFolder, err := getFolderRef(m, dataRoot, worldFolderName)
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
	defer func() {
		if err := m.Logout(); err != nil {
			log.WithError(err).Warn("Failed to logout from Mega")
		}
	}()

	worldsFolder, err := getCloudWorldsFolder(cfg, m)
	if err != nil {
		return nil, err
	}

	worlds, err := m.FS.GetChildren(worldsFolder)
	if err != nil {
		return nil, err
	}

	var result []WorldBackup
	for _, world := range worlds {
		if world.GetType() != mega.TypeFolder {
			continue
		}

		worldFolder, err := getFolderRef(m, worldsFolder, world.GetName())
		if err != nil {
			return nil, err
		}

		backups, err := m.FS.GetChildren(worldFolder)
		if err != nil {
			return nil, err
		}

		var generations []GenerationInfo
		for _, backup := range backups {
			name := backup.GetName()
			hash := backup.GetHash()
			timestamp := int(backup.GetTimeStamp().UnixMilli())
			if name[len(name)-7:] == ".tar.xz" {
				name = name[:len(name)-7]
			}

			if name != "5" {
				generations = append(generations, GenerationInfo{Gen: name, ID: hash, Timestamp: timestamp})
			}
		}

		if len(generations) == 0 {
			continue
		}

		sort.Slice(generations, func(i, j int) bool {
			if generations[i].Gen == "latest" {
				return true
			}
			if generations[j].Gen == "latest" {
				return false
			}
			iGen, err := strconv.Atoi(generations[i].Gen)
			if err != nil {
				return false
			}
			jGen, err := strconv.Atoi(generations[j].Gen)
			if err != nil {
				return false
			}
			return iGen < jGen
		})

		result = append(result, WorldBackup{
			WorldName:   world.GetName(),
			Generations: generations,
		})
	}

	return result, nil
}
