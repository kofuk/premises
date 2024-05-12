package rpc

import (
	"encoding/json"
	"errors"
	"io"
	"net"

	"github.com/kofuk/premises/runner/fs"
)

type Client struct {
	path string
}

var (
	ToExteriord      = NewClient(fs.DataPath("rpc@exteriord"))
	ToSnapshotHelper = NewClient(fs.DataPath("rpc@snapshot-helper"))
	ToLauncher       = NewClient(fs.DataPath("rpc@launcher"))
)

func NewClient(path string) *Client {
	return &Client{
		path: path,
	}
}

func readResponse(conn io.ReadWriter) (*Response[json.RawMessage], error) {
	body, err := readPacket(conn)
	if err != nil {
		return nil, err
	}

	var resp Response[json.RawMessage]
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, err
	}

	return &resp, nil
}

func handleCall(conn io.ReadWriter, method string, params, result any) error {
	reqID := 1
	if err := writePacket(conn, &Request[any]{
		Version: "2.0",
		ID:      &reqID,
		Method:  method,
		Params:  params,
	}); err != nil {
		return err
	}

	resp, err := readResponse(conn)
	if err != nil {
		return err
	}

	if resp.ID != reqID {
		return errors.New("invalid request id")
	}
	if resp.Error != nil {
		return resp.Error
	}

	if result == nil {
		return nil
	}

	if err := json.Unmarshal(resp.Result, result); err != nil {
		return err
	}

	return nil
}

func handleNotify(conn io.ReadWriter, method string, params any) error {
	return writePacket(conn, &Request[any]{
		Version: "2.0",
		Method:  method,
		Params:  params,
	})
}

func (c *Client) Call(method string, params, result any) error {
	conn, err := net.Dial("unix", c.path)
	if err != nil {
		return err
	}
	defer conn.Close()

	return handleCall(conn, method, params, result)
}

func (c *Client) Notify(method string, params any) error {
	conn, err := net.Dial("unix", c.path)
	if err != nil {
		return err
	}
	defer conn.Close()

	return handleNotify(conn, method, params)
}
