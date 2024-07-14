package conoha

import (
	_ "embed"
	"encoding/json"
	"fmt"

	"github.com/kofuk/premises/internal"
	"github.com/kofuk/premises/internal/entity/runner"
)

//go:embed startup.sh
var startupScriptTemplate string

func GenerateStartupScript(gameConfig *runner.Config) ([]byte, error) {
	gameConfigData, err := json.Marshal(gameConfig)
	if err != nil {
		return nil, err
	}

	return []byte(fmt.Sprintf(startupScriptTemplate, internal.ProtocolVersion, gameConfigData)), nil
}
