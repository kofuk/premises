package main

import (
	"os"

	"github.com/kofuk/premises/controlpanel/pmctl/user"
	"github.com/spf13/cobra"
)

func main() {
	cmd := &cobra.Command{
		Use: "pmctl",
	}

	cmd.AddCommand(user.NewUserCommand())

	err := cmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}
