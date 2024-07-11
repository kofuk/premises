package proxy

import (
	"context"
	"crypto/tls"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"math/rand"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/kofuk/premises/controlpanel/internal/config"
	"github.com/kofuk/premises/controlpanel/internal/kvs"
	"github.com/kofuk/premises/controlpanel/internal/longpoll"
	"github.com/kofuk/premises/internal/entity/runner"
	"github.com/kofuk/premises/internal/mc/protocol"
	"golang.org/x/sync/errgroup"
)

var ErrTimeout = errors.New("timeout")

type Connection struct {
	conn     io.ReadWriteCloser
	acquired bool
}

type ProxyHandler struct {
	kvs        kvs.KeyValueStore
	action     *longpoll.PollableActionService
	bindAddr   string
	iconURL    string
	gameDomain string
	cert       *Certificate
	pool       map[string]chan *Connection
	m          sync.Mutex
}

func NewProxyHandler(cfg *config.Config, kvs kvs.KeyValueStore, action *longpoll.PollableActionService) (*ProxyHandler, error) {
	bindAddr := cfg.ProxyBind
	if bindAddr == "" {
		bindAddr = "0.0.0.0:25565"
	}

	cert, err := generateCertificate()
	if err != nil {
		return nil, err
	}

	return &ProxyHandler{
		bindAddr:   bindAddr,
		kvs:        kvs,
		action:     action,
		iconURL:    cfg.IconURL,
		gameDomain: cfg.GameDomain,
		cert:       cert,
		pool:       make(map[string]chan *Connection),
	}, nil
}

func (p *ProxyHandler) startConnectorChannel(ctx context.Context) error {
	tcpListener, err := net.Listen("tcp", "0.0.0.0:25530")
	if err != nil {
		return err
	}
	keyPair, err := tls.X509KeyPair([]byte(p.cert.Cert), []byte(p.cert.Key))
	if err != nil {
		return err
	}

	listener := tls.NewListener(tcpListener, &tls.Config{
		Certificates: []tls.Certificate{keyPair},
	})

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		conn, err := listener.Accept()
		if err != nil {
			slog.Error("Error accepting connection", slog.Any("error", err))
			continue
		}

		go func() {
			defer conn.Close()

			buf := make([]byte, 36)
			n, err := conn.Read(buf)
			if err != nil {
				slog.Error("Error reading header", slog.Any("error", err))
				return
			} else if n != 36 || uuid.Validate(string(buf)) != nil {
				slog.Error("Invalid header")
				return
			}

			c := Connection{
				conn: conn,
			}

			p.m.Lock()
			ch := p.pool[string(buf)]
			if ch != nil {
				ch <- &c
			}
			p.m.Unlock()

			// If the connection is not handled within 30 seconds, close the it to avoid connection leak.
			time.Sleep(30 * time.Second)
			if !c.acquired {
				conn.Close()
			}
		}()
	}
}

func retrieveFavicon(url string) (string, error) {
	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	return "data:image/png;base64," + base64.RawStdEncoding.EncodeToString(data), nil
}

func (p *ProxyHandler) handleDummyServer(h *protocol.Handler, hs *protocol.Handshake) error {
	colors := []byte{'1', '2', '3', '4', '5', '6', '9', 'a', 'b', 'c', 'd', 'e', 'g'}
	color := colors[rand.Intn(len(colors))]

	status := protocol.Status{}
	status.Version.Name = "0.0.0+proxy"
	status.Version.Protocol = hs.Version
	status.Players.Max = 0
	status.Players.Online = 0
	status.Description.Text = fmt.Sprintf("§%[1]cServer §o\"%s\"§r§%[1]c is offline", color, hs.ServerAddr)
	status.EnforcesSecureChat = true

	if hs.ServerAddr == p.gameDomain && p.iconURL != "" {
		if favicon, err := retrieveFavicon(p.iconURL); err != nil {
			slog.Error("Error retrieving favicon", slog.Any("error", err))
		} else {
			status.Favicon = &favicon
		}
	}

	if err := h.ReadStatusRequest(); err != nil {
		return err
	}
	if err := h.WriteStatus(status); err != nil {
		return err
	}
	if err := h.HandlePingPong(); err != nil {
		return err
	}

	return nil
}

func (p *ProxyHandler) handleConn(conn io.ReadWriteCloser) error {
	defer conn.Close()

	if conn, ok := conn.(net.Conn); ok {
		if err := conn.SetDeadline(time.Now().Add(time.Minute)); err != nil {
			slog.Error("Error setting socket deadline", slog.Any("error", err))
		}
	}

	h := protocol.NewHandler(conn)

	hs, err := h.ReadHandshake()
	if err != nil {
		return fmt.Errorf("handshake error: %w", err)
	}

	if hs.ServerAddr != p.gameDomain {
		if hs.NextState != 1 {
			return fmt.Errorf("unknown server: %s", hs.ServerAddr)
		}

		return p.handleDummyServer(h, hs)
	}

	// TODO: Check if server is running

	connID := uuid.New()

	ch := make(chan *Connection)
	p.m.Lock()
	p.pool[connID.String()] = ch
	p.m.Unlock()

	deleteFromPool := func() {
		p.m.Lock()
		delete(p.pool, connID.String())
		p.m.Unlock()
	}
	defer deleteFromPool()

	p.action.Push(context.TODO(), "default", runner.Action{
		Type: runner.ActionConnReq,
		ConnReq: &runner.ConnReqInfo{
			ConnectionID: connID.String(),
			ServerCert:   p.cert.Cert,
		},
	})

	timer := time.NewTimer(5 * time.Second)
	var upstrm io.ReadWriteCloser
	defer timer.Stop()
	select {
	case c := <-ch:
		upstrm = c.conn
		c.acquired = true

	case <-timer.C:
		if hs.NextState != 1 {
			return fmt.Errorf("connector not responded within 5 seconds: %s", hs.ServerAddr)
		}

		return p.handleDummyServer(h, hs)
	}
	deleteFromPool()

	if conn, ok := conn.(net.Conn); ok {
		// Unset deadline, because the connection is handled by the upstream server.
		if err := conn.SetDeadline(time.Time{}); err != nil {
			slog.Error("Error setting socket deadline", slog.Any("error", err))
		}
	}

	var eg errgroup.Group

	eg.Go(func() error {
		upstrm.Write(h.OrigBytes())

		io.Copy(upstrm, conn)
		upstrm.Close()
		conn.Close()
		return nil
	})
	eg.Go(func() error {
		io.Copy(conn, upstrm)
		upstrm.Close()
		conn.Close()
		return nil
	})

	return eg.Wait()
}

func (p *ProxyHandler) startProxy(ctx context.Context, addr string) error {
	l, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	defer l.Close()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		conn, err := l.Accept()
		if err != nil {
			slog.Error("Error accepting connection", slog.Any("error", err))
			continue
		}

		// Handle connection asynchronously
		go func() {
			defer func() {
				if err := recover(); err != nil {
					slog.Error("Error in handler (error recovered)", slog.Any("error", err))
				}
			}()

			if err := p.handleConn(conn); err != nil {
				slog.Error("Error handling connection", slog.Any("error", err))
			}
			slog.Debug("Connection closed")
		}()
	}
}

func (p *ProxyHandler) Start(ctx context.Context) error {
	eg, ctx := errgroup.WithContext(ctx)
	eg.Go(func() error {
		return p.startConnectorChannel(ctx)
	})
	eg.Go(func() error {
		return p.startProxy(ctx, p.bindAddr)
	})

	return eg.Wait()
}
