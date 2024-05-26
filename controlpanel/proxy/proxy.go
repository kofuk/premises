package proxy

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"math/rand"
	"net"
	"net/http"
	"time"

	"github.com/kofuk/premises/controlpanel/config"
	"github.com/kofuk/premises/controlpanel/kvs"
	"github.com/kofuk/premises/internal/mc/protocol"
	"github.com/labstack/echo/v4"
	"golang.org/x/sync/errgroup"
)

type ProxyHandler struct {
	kvs        kvs.KeyValueStore
	bindAddr   string
	iconURL    string
	gameDomain string
}

func NewProxyHandler(cfg *config.Config, kvs kvs.KeyValueStore) *ProxyHandler {
	bindAddr := cfg.ProxyBind
	if bindAddr == "" {
		bindAddr = "0.0.0.0:25565"
	}

	return &ProxyHandler{
		bindAddr:   bindAddr,
		kvs:        kvs,
		iconURL:    cfg.IconURL,
		gameDomain: cfg.GameDomain,
	}
}

func (p *ProxyHandler) startInternalApi(ctx context.Context) error {
	e := echo.New()
	e.HideBanner = true
	e.HidePort = true

	e.POST("/set", func(c echo.Context) error {
		name := c.QueryParam("name")
		addr := c.QueryParam("addr")

		slog.Info("Setting proxy host", slog.String("name", name), slog.String("addr", addr))

		if err := p.kvs.Set(ctx, "proxy:"+name, addr, -1); err != nil {
			slog.Error("Error setting proxy host", slog.Any("error", err))
		}

		return c.String(http.StatusOK, "success")
	})

	e.POST("/clear", func(c echo.Context) error {
		name := c.QueryParam("name")

		slog.Info("Removing proxy host", slog.String("name", name))

		if err := p.kvs.Del(ctx, "proxy:"+name); err != nil {
			slog.Error("Error removing proxy host", slog.Any("error", err))
		}

		return c.String(http.StatusNoContent, "success")
	})

	go func() {
		<-ctx.Done()
		e.Close()
	}()
	return e.Start(":8001")
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

	var addr string
	if err := p.kvs.Get(context.TODO(), "proxy:"+hs.ServerAddr, &addr); err != nil {
		slog.Debug("Error getting proxy host", slog.Any("error", err))
	}

	if addr == "" {
		if hs.NextState != 1 {
			return fmt.Errorf("unknown server: %s", hs.ServerAddr)
		}

		return p.handleDummyServer(h, hs)
	}

	upstrm, err := net.Dial("tcp", addr)
	if err != nil {
		// Connection error. We'll respond with dummy response (if possible).
		if hs.NextState != 1 {
			return err
		}

		if err2 := p.handleDummyServer(h, hs); err2 != nil {
			return errors.Join(err, err2)
		}
		return err
	}

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
		return p.startInternalApi(ctx)
	})
	eg.Go(func() error {
		return p.startProxy(ctx, p.bindAddr)
	})

	return eg.Wait()
}
