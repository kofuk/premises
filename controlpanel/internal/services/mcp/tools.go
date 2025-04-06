package mcp

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/kofuk/premises/controlpanel/internal/launcher"
	"github.com/mark3labs/mcp-go/mcp"
)

type ToolHandler struct {
	server *MCPServer
}

func toJSON(v any) string {
	json, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return ""
	}
	return string(json)
}

func (*ToolHandler) ListHardwareConfigurations(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return mcp.NewToolResultText(`- "2g
    - A small server with 2 GB of RAM and 3-core CPU.
	- It costs 3.3 JPY per hour.
	- Suitable for small prototypes and testing worlds.
- "4g"
    - A moderate server with 4 GB of RAM and 4-core CPU.
	- It costs 6.6 JPY per hour.
	- Suitable for moderate worlds with 4-6 players.
- "8g"
    - A large server with 8 GB of RAM and 6-core CPU.
	- It costs 13.2 JPY per hour.
	- Suitable for large and complex worlds.
- "16g"
	- A very large server with 16 GB of RAM and 8-core CPU.
	- It costs 24.2 JPY per hour.
	- Suitable for very large worlds with many players.
- "32g"
	- A massive server with 32 GB of RAM and 12-core CPU.
	- It costs 48.0 JPY per hour.
	- You will NOT need this machine as far as you are playing vanilla Minecraft.
- "64g"
	- An enormous server with 64 GB of RAM and 16-core CPU.
	- It costs 96.8 JPY per hour.
	- You will NOT need this machine as far as you are playing vanilla Minecraft.`), nil
}

func (t *ToolHandler) ListExistingWorlds(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	worlds, err := t.server.worldService.GetWorlds(ctx)
	if err != nil {
		return nil, err
	}
	worldNames := make([]string, 0)
	for _, world := range worlds {
		worldNames = append(worldNames, world.WorldName)
	}

	return mcp.NewToolResultText(toJSON(worldNames)), nil
}

func (t *ToolHandler) LaunchServer(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	machineType, ok := req.Params.Arguments["machine_type"].(string)
	if !ok {
		return nil, errors.New("machine_type is required")
	}
	worldName, ok := req.Params.Arguments["world_name"].(string)
	if !ok {
		return nil, errors.New("world_name is required")
	}

	minecraftVersion, err := t.server.mcVersionsService.GetLatestRelease(ctx)
	if err != nil {
		return nil, err
	}

	serverInfo, err := t.server.mcVersionsService.GetServerInfo(ctx, minecraftVersion)
	if err != nil {
		return nil, err
	}

	err = t.server.launcherService.Launch(ctx, &launcher.LaunchConfig{
		MachineType: machineType,
		Server: launcher.LaunchServerConfig{
			PreferDetected:   true,
			Version:          minecraftVersion,
			DownloadUrl:      serverInfo.DownloadURL,
			ManifestOverride: t.server.mcVersionsService.GetOverridenManifestURL(),
			CustomCommand:    serverInfo.LaunchCommand,
			JavaVersion:      serverInfo.JavaVersion,
			InactiveTimeout:  -1,
			Motd:             "",
			Operators:        t.server.operators,
			Whitelist:        t.server.whitelist,
		},
		World: launcher.LaunchWorldConfig{
			ShouldGenerate: false,
			Name:           worldName,
			GenerationID:   "@/latest",
			Seed:           "",
			LevelType:      "default",
			Difficulty:     "easy",
		},
	})
	if err != nil {
		return nil, err
	}

	return mcp.NewToolResultText("Server launched successfully."), nil
}
