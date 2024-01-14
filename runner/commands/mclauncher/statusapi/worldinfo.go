package statusapi

import (
	"github.com/kofuk/premises/common/entity/runner"
	"github.com/kofuk/premises/runner/commands/mclauncher/gamesrv"
)

type WorldInfo struct {
	ServerVersion string `json:"serverVersion"`
	World         struct {
		Name string `json:"name"`
		Seed string `json:"seed"`
	} `json:"world"`
}

func GetWorldInfo(config *runner.Config, srv *gamesrv.ServerInstance) (*WorldInfo, error) {
	result := &WorldInfo{}
	result.ServerVersion = config.Server.Version
	result.World.Name = config.World.Name
	seed, err := srv.GetSeed()
	if err != nil {
		return nil, err
	}
	result.World.Seed = seed

	return result, nil
}
