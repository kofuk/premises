package serverproperties

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"regexp"
	"slices"
	"strings"
)

var defaultServerProperties = map[string]string{
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
	"query.port":                        "32109",
	"rate-limit":                        "0",
	"rcon.password":                     "x",
	"rcon.port":                         "25575",
	"require-resource-pack":             "false",
	"resource-pack":                     "",
	"resource-pack-prompt":              "",
	"resource-pack-sha1":                "",
	"server-ip":                         "127.0.0.1",
	"server-port":                       "32109",
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

var keyRegexp = regexp.MustCompile(`^[a-zA-Z0-9._-]+$`)

type ServerPropertiesGenerator struct {
	properties map[string]string
}

func NewServerPropertiesGenerator() *ServerPropertiesGenerator {
	properties := make(map[string]string)
	for k, v := range defaultServerProperties {
		properties[k] = v
	}

	return &ServerPropertiesGenerator{
		properties: properties,
	}
}

func (g *ServerPropertiesGenerator) SetMotd(motd string) error {
	return g.Set("motd", motd)
}

func (g *ServerPropertiesGenerator) SetDifficulty(difficulty string) error {
	if !slices.Contains([]string{"peaceful", "easy", "normal", "hard"}, difficulty) {
		return errors.New("unknown difficulty")
	}

	return g.Set("difficulty", difficulty)
}

func (g *ServerPropertiesGenerator) SetLevelType(levelType string) error {
	if !slices.Contains([]string{"flat", "largebiomes", "amplified", "default"}, levelType) {
		return errors.New("unknown level type")
	}

	return g.Set("level-type", levelType)
}

func (g *ServerPropertiesGenerator) SetSeed(seed string) error {
	return g.Set("level-seed", seed)
}

func (g *ServerPropertiesGenerator) Set(key, value string) error {
	if _, ok := overrideBlockedProps[key]; ok {
		return errors.New("this property is not allowed to be overridden")
	}
	if !keyRegexp.MatchString(key) {
		return errors.New("invalid property key")
	}

	sanitizedValue := strings.ReplaceAll(strings.ReplaceAll(value, "\r", ""), "\n", " ")
	g.properties[key] = sanitizedValue

	return nil
}

func (g *ServerPropertiesGenerator) Write(w io.Writer) error {
	writer := bufio.NewWriter(w)
	defer writer.Flush()

	for key, value := range g.properties {
		escapedValue := strings.ReplaceAll(value, "\\", "\\\\")
		fmt.Fprintf(writer, "%s=%s\n", key, escapedValue)
	}

	return nil
}
