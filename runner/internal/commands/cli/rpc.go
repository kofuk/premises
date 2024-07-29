package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/kofuk/premises/runner/internal/rpc"
)

type RPC struct {
}

func (r *RPC) Run(args []string) int {
	if len(args) < 3 {
		fmt.Fprintln(os.Stderr, `usage: premises-runner cli rpc <path> <call|notify> <method>
Read params from stdin and send a request to the RPC server`)
		return 1
	}
	path := args[0]
	reqType := args[1]
	method := args[2]

	data, err := io.ReadAll(os.Stdin)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Unable to read from stdin:", err.Error())
		return 1
	}

	client := rpc.NewClient(path)
	if reqType == "call" {
		var resp json.RawMessage
		if err := client.Call(method, json.RawMessage(data), &resp); err != nil {
			fmt.Fprintln(os.Stderr, "Error:", err.Error())
			return 1
		}
		fmt.Println(string(resp))
	} else if reqType == "notify" {
		if err := client.Notify(method, json.RawMessage(data)); err != nil {
			fmt.Fprintln(os.Stderr, "Error:", err.Error())
			return 1
		}
	} else {
		fmt.Fprintln(os.Stderr, "Unknown request type")
		return 1
	}

	return 0
}
