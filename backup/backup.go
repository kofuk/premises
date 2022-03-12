package backup

import (
	"errors"
	"regexp"
	"sort"

	"github.com/kofuk/premises/config"
	"github.com/t3rm1n4l/go-mega"
)

type WorldBackup struct {
	WorldName   string   `json:"worldName"`
	Generations []string `json:"generations"`
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

	worlds, err := m.FS.GetChildren(worldsFolder)
	if err != nil {
		return nil, err
	}

	var result []WorldBackup
	for _, world := range worlds {
		if world.GetType() != mega.FOLDER {
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

		var generations []string
		for _, backup := range backups {
			name := backup.GetName()
			if name[len(name)-7:] == ".tar.xz" {
				name = name[:len(name)-7]
			}

			generations = append(generations, name)
		}

		if len(generations) == 0 {
			continue
		}

		sort.Strings(generations)
		// "latest" should be the first.
		if len(generations) > 1 && generations[len(generations)-1] == "latest" {
			generations = append([]string{"latest"}, generations[:len(generations)-1]...)
		}

		result = append(result, WorldBackup{
			WorldName:   world.GetName(),
			Generations: generations,
		})
	}

	return result, nil
}
