package rpc

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net"
)

type HandlerFunc func(req *AbstractRequest) (any, error)

type Server struct {
	path    string
	methods map[string]HandlerFunc
}

func NewServer(path string) *Server {
	return &Server{
		path:    path,
		methods: make(map[string]HandlerFunc),
	}
}

func (s *Server) RegisterMethod(name string, fn HandlerFunc) {
	s.methods[name] = fn
}

func readRequest[T any](r io.Reader) (*Request[T], error) {
	body, err := readPacket(r)
	if err != nil {
		return nil, err
	}

	var req Request[T]
	if err := json.Unmarshal(body, &req); err != nil {
		return nil, err
	}

	return &req, nil
}

func writeResponse[T any](w io.Writer, data *Response[T]) error {
	if err := writePacket(w, data); err != nil {
		return err
	}

	return nil
}

func (s *Server) handleRequest(req *AbstractRequest) *Response[any] {
	if req.Version != "2.0" {
		return &Response[any]{
			Version: "2.0",
			ID:      req.ID,
			Error: &RPCError{
				Code:    InvalidRequest,
				Message: InvalidRequestMessage,
			},
		}
	}

	method, ok := s.methods[req.Method]
	if !ok {
		return &Response[any]{
			Version: "2.0",
			ID:      req.ID,
			Error: &RPCError{
				Code:    MethodNotFound,
				Message: MethodNotFoundMessage,
			},
		}
	}

	result, err := method(req)
	if err != nil {
		return &Response[any]{
			Version: "2.0",
			ID:      req.ID,
			Error: &RPCError{
				Code:    CallerError,
				Message: ServerErrorMessage,
				Data:    err.Error(),
			},
		}
	}

	return &Response[any]{
		Version: "2.0",
		ID:      req.ID,
		Result:  result,
	}
}

func (s *Server) handleConnection(conn io.ReadWriteCloser) error {
	defer conn.Close()

	req, err := readRequest[json.RawMessage](conn)
	if err != nil {
		return err
	}

	resp := s.handleRequest((*AbstractRequest)(req))

	if err := writeResponse(conn, resp); err != nil {
		return err
	}

	return nil
}

func (s *Server) Start(ctx context.Context) error {
	l, err := net.Listen("unix", s.path)
	if err != nil {
		return err
	}

	go func() {
		<-ctx.Done()
		l.Close()
	}()

	for {
		conn, err := l.Accept()
		if err != nil {
			slog.Error(err.Error())
			continue
		}

		go func() {
			if err := s.handleConnection(conn); err != nil {
				slog.Error(err.Error())
			}
		}()
	}
}
