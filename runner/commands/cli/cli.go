package cli

import (
	"fmt"
	"os"
)

func Run(args []string) int {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, `Subcommand required.
Available subcommands:
  rcon [host:port] [password]
    Minecraft RCON client`)
		return 1
	}

	switch args[0] {
	case "rcon":
		rcon := new(Rcon)
		return rcon.Run(args[1:])
	}

	fmt.Fprintln(os.Stderr, "Unknown subcommand")
	return 1
}
