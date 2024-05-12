package rpc

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net"
	"os"
	"sync"
)

type HandlerFunc func(req *AbstractRequest) (any, error)
type NotifyHandlerFunc func(req *AbstractRequest) error

type Server struct {
	path          string
	methods       map[string]HandlerFunc
	notifyMethods map[string]NotifyHandlerFunc
	m             sync.Mutex
}

func NewServer(path string) *Server {
	return &Server{
		path:          path,
		methods:       make(map[string]HandlerFunc),
		notifyMethods: make(map[string]NotifyHandlerFunc),
	}
}

var DefaultServer *Server

func InitializeDefaultServer(path string) {
	DefaultServer = NewServer(path)
}

func (s *Server) RegisterMethod(name string, fn HandlerFunc) {
	s.m.Lock()
	defer s.m.Unlock()
	s.methods[name] = fn
}

func (s *Server) RegisterNotifyMethod(name string, fn NotifyHandlerFunc) {
	s.m.Lock()
	defer s.m.Unlock()
	s.notifyMethods[name] = fn
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

func (s *Server) getMethod(name string) (HandlerFunc, bool) {
	s.m.Lock()
	defer s.m.Unlock()
	fn, ok := s.methods[name]
	return fn, ok
}

func (s *Server) getNotifyMethod(name string) (NotifyHandlerFunc, bool) {
	s.m.Lock()
	defer s.m.Unlock()
	fn, ok := s.notifyMethods[name]
	return fn, ok
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

	if req.ID == 0 {
		// notify
		method, ok := s.getNotifyMethod(req.Method)
		if !ok {
			slog.Error("Method for notify request not found", slog.String("method", req.Method))
			return nil
		}
		if err := method(req); err != nil {
			slog.Error("Error handling notification", slog.Any("error", err))
			return nil
		}
		return nil
	}

	// method call
	method, ok := s.getMethod(req.Method)
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

	if resp != nil {
		if err := writeResponse(conn, resp); err != nil {
			return err
		}
	}

	return nil
}

func (s *Server) Start(ctx context.Context) error {
	os.Remove(s.path)

	l, err := net.Listen("unix", s.path)
	if err != nil {
		return err
	}
	defer os.Remove(s.path)

	os.Chmod(s.path, 0666)

	go func() {
		<-ctx.Done()
		l.Close()
	}()

	for {
		conn, err := l.Accept()
		if err != nil {
			if errors.Is(err, net.ErrClosed) {
				return nil
			}

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
