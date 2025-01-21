package cli

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

func Run(ctx context.Context, args []string) int {
	cmd := &cobra.Command{
		Use: "premises-runner-cli",
	}
	cmd.SetArgs(args)
	cmd.AddCommand(
		NewRconCommand(),
		NewRPCCommand(),
	)

	if err := cmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}

	return 0
}
