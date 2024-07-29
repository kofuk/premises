package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/kofuk/premises/runner/internal/rpc"
	"github.com/spf13/cobra"
)

type RPC struct {
	Path        string
	RequestType string
	Method      string
}

func NewRPCCommand() *cobra.Command {
	rpc := &RPC{}

	cmd := &cobra.Command{
		Use:   "rpc",
		Short: "Send a request to the RPC server",
		RunE: func(cmd *cobra.Command, args []string) error {
			return rpc.Run()
		},
	}

	flags := cmd.Flags()

	flags.StringVarP(&rpc.Path, "path", "p", "", "Path to the RPC server")
	flags.StringVarP(&rpc.RequestType, "type", "t", "call", "Request type (call or notify)")
	flags.StringVarP(&rpc.Method, "method", "m", "", "Method to call")

	return cmd
}

func (r *RPC) Run() error {
	data, err := io.ReadAll(os.Stdin)
	if err != nil {
		return fmt.Errorf("failed to read stdin: %w", err)
	}

	client := rpc.NewClient(r.Path)

	switch r.RequestType {
	case "call":
		var resp json.RawMessage
		if err := client.Call(r.Method, json.RawMessage(data), &resp); err != nil {
			return err
		}
		fmt.Println(string(resp))

	case "notify":
		if err := client.Notify(r.Method, json.RawMessage(data)); err != nil {
			return err
		}

	default:
		return errors.New("unknown request type")
	}

	return nil
}
