package statusapi

import (
	"github.com/kofuk/premises/runner/commands/mclauncher/config"
	"github.com/kofuk/premises/runner/commands/mclauncher/gamesrv"
)

type WorldInfo struct {
	ServerVersion string `json:"serverVersion"`
	World         struct {
		Name string `json:"name"`
		Seed string `json:"seed"`
	} `json:"world"`
}

func GetWorldInfo(ctx *config.PMCMContext, srv *gamesrv.ServerInstance) (*WorldInfo, error) {
	result := &WorldInfo{}
	result.ServerVersion = ctx.Cfg.Server.Version
	result.World.Name = ctx.Cfg.World.Name
	seed, err := srv.GetSeed()
	if err != nil {
		return nil, err
	}
	result.World.Seed = seed

	return result, nil
}
