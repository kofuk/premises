package main

import (
	"os"

	admincli "github.com/kofuk/premises/backend/services/pmctl/commands"
	"github.com/spf13/cobra"
)

func main() {
	cmd := &cobra.Command{
		Use: "pmctl",
	}

	cmd.AddCommand(admincli.NewUserCommand())
	cmd.AddCommand(admincli.NewCopyStaticCommand())

	err := cmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}
