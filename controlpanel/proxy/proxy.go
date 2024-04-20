package proxy

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"math/rand"
	"net"
	"net/http"
	"os"
	"sync"

	"github.com/kofuk/premises/common/mc/protocol"
	"github.com/labstack/echo/v4"
	"golang.org/x/sync/errgroup"
)

type ProxyHandler struct {
	m        sync.Mutex
	bindAddr string
	servers  map[string]string
}

func NewProxyHandler() *ProxyHandler {
	bindAddr := os.Getenv("PREMISES_PROXY_BIND")
	if bindAddr == "" {
		bindAddr = "0.0.0.0:25565"
	}

	return &ProxyHandler{
		bindAddr: bindAddr,
		servers:  make(map[string]string),
	}
}

func (p *ProxyHandler) startInternalApi(ctx context.Context) error {
	e := echo.New()
	e.HideBanner = true
	e.HidePort = true

	e.POST("/set", func(c echo.Context) error {
		p.m.Lock()
		defer p.m.Unlock()

		name := c.QueryParam("name")
		addr := c.QueryParam("addr")
		p.servers[name] = addr

		return c.String(http.StatusOK, "success")
	})

	e.POST("/clear", func(c echo.Context) error {
		p.m.Lock()
		defer p.m.Unlock()

		name := c.QueryParam("name")

		delete(p.servers, name)

		return c.String(http.StatusNoContent, "success")
	})

	go func() {
		<-ctx.Done()
		e.Close()
	}()
	return e.Start(":8001")
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

	h := protocol.NewHandler(conn)

	hs, err := h.ReadHandshake()
	if err != nil {
		return errors.New("Handshake error")
	}

	p.m.Lock()
	addr, ok := p.servers[hs.ServerAddr]
	p.m.Unlock()

	if !ok {
		if hs.NextState != 1 {
			return fmt.Errorf("Unknown server: %s", hs.ServerAddr)
		}

		return p.handleDummyServer(h, hs)
	}

	upstrm, err := net.Dial("tcp", addr)
	if err != nil {
		return err
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
		return p.startInternalApi(ctx)
	})
	eg.Go(func() error {
		return p.startProxy(ctx, p.bindAddr)
	})

	return eg.Wait()
}
