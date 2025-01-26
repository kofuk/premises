package rpc

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/kofuk/premises/internal/otel"
)

const (
	ParseError            = -32700
	ParseErrorMessage     = "Parse error"
	InvalidRequest        = -32600
	InvalidRequestMessage = "Invalid request"
	MethodNotFound        = -32601
	MethodNotFoundMessage = "Method not found"
	InvalidParams         = -32602
	InvalidParamsMessage  = "Invalid params"
	InternalError         = -32603
	InternalErrorMessage  = "Internal error"
	// Implementation-defined errors
	CallerError        = -32000
	ServerErrorMessage = "Server error"
)

type Request[T any] struct {
	Version     string `json:"jsonrpc"`
	ID          *int   `json:"id,omitempty"`
	Method      string `json:"method"`
	Params      T      `json:"params"`
	Traceparent string `json:"-"`
}

type AbstractRequest Request[json.RawMessage]

func (req *AbstractRequest) Bind(v any) error {
	return json.Unmarshal(req.Params, v)
}

type RPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    string `json:"data,omitempty"`
}

func (e *RPCError) Error() string {
	msg := fmt.Sprintf("RPCError: %d: %s", e.Code, e.Message)
	if e.Data != "" {
		msg += ": " + e.Data
	}
	return msg
}

type Response[T any] struct {
	Version     string    `json:"jsonrpc"`
	ID          int       `json:"id"`
	Result      T         `json:"result,omitempty"`
	Error       *RPCError `json:"error,omitempty"`
	Traceparent string    `json:"-"`
}

type Packet struct {
	ContentLength int
	Traceparent   string
	Body          json.RawMessage
}

func readPacket(r io.Reader) (*Packet, error) {
	br := bufio.NewReader(r)
	packet := &Packet{
		ContentLength: -1,
	}
	for {
		l, _, err := br.ReadLine()
		if err != nil {
			return nil, err
		}
		if len(l) == 0 {
			break
		}
		f := strings.SplitN(string(l), ": ", 2)
		if len(f) != 2 {
			return nil, errors.New("invalid header")
		}
		if strings.EqualFold(f[0], "content-length") {
			packet.ContentLength, err = strconv.Atoi(f[1])
			if err != nil {
				return nil, err
			}
		} else if strings.EqualFold(f[0], "traceparent") {
			packet.Traceparent = f[1]
		}
	}

	if packet.ContentLength < 0 {
		return nil, errors.New("invalid length")
	}

	buf := make([]byte, packet.ContentLength)
	if _, err := io.ReadFull(br, buf); err != nil {
		return nil, err
	}

	var body json.RawMessage
	if err := json.Unmarshal(buf, &body); err != nil {
		return nil, err
	}

	packet.Body = body

	return packet, nil
}

func writePacket(ctx context.Context, w io.Writer, body any) error {
	bw := bufio.NewWriter(w)
	defer bw.Flush()

	bodyJSON, err := json.Marshal(&body)
	if err != nil {
		return err
	}

	bw.WriteString(fmt.Sprintf("Content-Length: %d\r\n", len(bodyJSON)))
	if traceparent := otel.TraceContextFromContext(ctx); traceparent != "" {
		bw.WriteString(fmt.Sprintf("Traceparent: %s\r\n", traceparent))
	}
	bw.WriteString("\r\n")
	bw.Write(bodyJSON)

	return nil
}
