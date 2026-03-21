package cli

import (
	"context"
	"fmt"
	"os"

	"github.com/kofuk/premises/backend/common/entity/runner"
	"github.com/spf13/cobra"
)

func Run(ctx context.Context, config *runner.Config, args []string) int {
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
