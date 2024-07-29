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
    Minecraft RCON client
  rpc <path> <call|notify> <method>
    Send a request to the RPC server`)
		return 1
	}

	switch args[0] {
	case "rcon":
		rcon := new(Rcon)
		return rcon.Run(args[1:])
	case "rpc":
		rpc := new(RPC)
		return rpc.Run(args[1:])
	}

	fmt.Fprintln(os.Stderr, "Unknown subcommand")
	return 1
}
