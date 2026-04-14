package startup

import (
	_ "embed"
	"encoding/json"
	"fmt"

	"github.com/kofuk/premises/backend/common"
	"github.com/kofuk/premises/backend/common/entity/runner"
)

//go:embed startup.sh
var startupScriptTemplate string

func GenerateStartupScript(gameConfig *runner.Config) ([]byte, error) {
	gameConfigData, err := json.Marshal(gameConfig)
	if err != nil {
		return nil, err
	}

	var result []byte
	result = fmt.Appendf(result, startupScriptTemplate, common.Version, gameConfigData)
	return result, nil
}
