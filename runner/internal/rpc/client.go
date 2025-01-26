package rpc

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net"

	"github.com/kofuk/premises/runner/internal/env"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

type Client struct {
	path string
}

var (
	ToExteriord      = NewClient(env.DataPath("rpc@exteriord"))
	ToSnapshotHelper = NewClient(env.DataPath("rpc@snapshot-helper"))
	ToLauncher       = NewClient(env.DataPath("rpc@launcher"))
	ToConnector      = NewClient(env.DataPath("rpc@connector"))
)

func NewClient(path string) *Client {
	return &Client{
		path: path,
	}
}

func readResponse(conn io.ReadWriter) (*Response[json.RawMessage], error) {
	packet, err := readPacket(conn)
	if err != nil {
		return nil, err
	}

	var resp Response[json.RawMessage]
	if err := json.Unmarshal(packet.Body, &resp); err != nil {
		return nil, err
	}

	resp.Traceparent = packet.Traceparent

	return &resp, nil
}

func handleCall(ctx context.Context, conn io.ReadWriter, method string, params, result any) error {
	reqID := 1
	if err := writePacket(ctx, conn, &Request[any]{
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

func handleNotify(ctx context.Context, conn io.ReadWriter, method string, params any) error {
	return writePacket(ctx, conn, &Request[any]{
		Version: "2.0",
		Method:  method,
		Params:  params,
	})
}

func (c *Client) Call(ctx context.Context, method string, params, result any) error {
	ctx, span := tracer.Start(ctx, "RPC call",
		trace.WithSpanKind(trace.SpanKindClient),
		trace.WithAttributes(attribute.String("rpc.method", method)),
	)
	defer span.End()

	conn, err := net.Dial("unix", c.path)
	if err != nil {
		return err
	}
	defer conn.Close()

	err = handleCall(ctx, conn, method, params, result)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
	}

	return err
}

func (c *Client) Notify(ctx context.Context, method string, params any) error {
	ctx, span := tracer.Start(ctx, "RPC notify",
		trace.WithSpanKind(trace.SpanKindClient),
		trace.WithAttributes(attribute.String("rpc.method", method)),
	)
	defer span.End()

	conn, err := net.Dial("unix", c.path)
	if err != nil {
		return err
	}
	defer conn.Close()

	err = handleNotify(ctx, conn, method, params)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
	}

	return err
}
