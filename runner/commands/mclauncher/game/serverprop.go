package game

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"slices"
	"strings"

	"github.com/kofuk/premises/common/entity/runner"
)

var serverProperties = map[string]string{
	"allow-flight":                      "false",
	"allow-nether":                      "true",
	"broadcast-console-to-ops":          "false",
	"broadcast-rcon-to-ops":             "false",
	"difficulty":                        "easy",
	"enable-command-block":              "true",
	"enable-jmx-monitoring":             "false",
	"enable-query":                      "false",
	"enable-rcon":                       "true",
	"enable-status":                     "true",
	"enforce-whitelist":                 "true",
	"entity-broadcast-range-percentage": "100",
	"force-gamemode":                    "false",
	"function-permission-level":         "2",
	"gamemode":                          "survival",
	"hardcore":                          "false",
	"hide-online-players":               "false",
	"level-name":                        "world",
	"max-players":                       "20",
	"max-tick-time":                     "60000",
	"max-world-size":                    "29999984",
	"motd":                              "",
	"network-compression-threshold":     "256",
	"online-mode":                       "true",
	"op-permission-level":               "4",
	"player-idle-timeout":               "0",
	"prevent-proxy-connections":         "false",
	"pvp":                               "true",
	"query.port":                        "25565",
	"rate-limit":                        "0",
	"rcon.password":                     "x",
	"rcon.port":                         "25575",
	"require-resource-pack":             "false",
	"resource-pack":                     "",
	"resource-pack-prompt":              "",
	"resource-pack-sha1":                "",
	"server-ip":                         "",
	"server-port":                       "25565",
	"simulation-distance":               "10",
	"spawn-animals":                     "true",
	"spawn-monsters":                    "true",
	"spawn-npcs":                        "true",
	"spawn-protection":                  "0",
	"sync-chunk-writes":                 "true",
	"text-filtering-config":             "",
	"use-native-transport":              "true",
	"view-distance":                     "10",
	"white-list":                        "true",
}

// These properties are not allowed users to override by configuration, because it
// 1. breaks environment which runner assumes
// 2. unsafe for public server to change value
var overrideBlockedProps = map[string]struct{}{
	"enable-jmx-monitoring": {},
	"enable-query":          {},
	"enable-rcon":           {},
	"level-name":            {},
	"rcon.password":         {},
	"rcon.port":             {},
	"server-ip":             {},
	"server-port":           {},
	"white-list":            {},
}

type ServerProperties struct {
	props map[string]string
}

func NewServerProperties() *ServerProperties {
	return &ServerProperties{
		props: serverProperties,
	}
}

func (p *ServerProperties) LoadConfig(config *runner.Config) error {
	serverProps := NewServerProperties()
	serverProps.SetMotd(config.Motd)
	serverProps.SetDifficulty(config.World.Difficulty)
	serverProps.SetLevelType(config.World.LevelType)
	serverProps.SetSeed(config.World.Seed)
	serverProps.OverrideProperties(config.Server.ServerPropOverride)
	return nil
}

func (p *ServerProperties) SetMotd(motd string) {
	p.props["motd"] = strings.ReplaceAll(strings.ReplaceAll(motd, "\r", ""), "\n", " ")
}

func (p *ServerProperties) SetDifficulty(difficulty string) error {
	if !slices.Contains([]string{"peaceful", "easy", "normal", "hard"}, difficulty) {
		return errors.New("unknown difficulty")
	}
	p.props["difficulty"] = difficulty
	return nil
}

func (p *ServerProperties) SetLevelType(levelType string) error {
	if !slices.Contains([]string{"default", "flat", "largeBiomes", "amplified", "buffet"}, levelType) {
		return errors.New("unknown world type")
	}
	p.props["level-type"] = levelType
	return nil
}

func (p *ServerProperties) OverrideProperties(props map[string]string) {
	for k, v := range props {
		if strings.Trim(k, "abcdefghijklmnopqrstuvwxyz0123456789.-") != "" {
			continue
		}
		if _, ok := overrideBlockedProps[k]; ok {
			continue
		}
		p.props[k] = v
	}
}

func (p *ServerProperties) SetSeed(seed string) {
	p.props["level-seed"] = seed
}

func (p *ServerProperties) DangerouslySetProperty(k, v string) {
	p.props[k] = v
}

func (p *ServerProperties) Write(out io.Writer) error {
	writer := bufio.NewWriter(out)
	defer writer.Flush()

	for k, v := range p.props {
		escapedValue := strings.ReplaceAll(v, "\\", "\\\\")
		fmt.Fprintf(writer, "%s=%s\n", k, escapedValue)
	}

	return nil
}
