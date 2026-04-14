package connector

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"io"
	"net"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

type Proxy struct {
	ID       string
	Endpoint string
	Cert     string
	Metrics  *Metrics
}

type peer string

const (
	peerServer peer = "server"
	peerClient peer = "client"
)

type connection struct {
	io.ReadWriteCloser
	peerKind peer
}

func createTLSConfig(cert string) (*tls.Config, error) {
	rootCAs := x509.NewCertPool()
	if ok := rootCAs.AppendCertsFromPEM([]byte(cert)); !ok {
		return nil, errors.New("error appending certificate")
	}
	return &tls.Config{
		RootCAs:    rootCAs,
		ServerName: "control.fake.premises.kofuk.org",
	}, nil
}

func (p *Proxy) copyWithMeter(ctx context.Context, dst connection, src connection) {
	buf := make([]byte, 32*1024)
	for {
		n, err := src.Read(buf)
		if n > 0 {
			if _, err := dst.Write(buf[:n]); err != nil {
				return
			}
			p.Metrics.io.Add(ctx, int64(n), metric.WithAttributes(
				attribute.String("from", string(src.peerKind)),
				attribute.String("to", string(dst.peerKind)),
			))
		}
		if err != nil {
			if err != io.EOF {
				return
			}
			break
		}
	}
}

func (p *Proxy) Run(ctx context.Context) error {
	tlsConfig, err := createTLSConfig(p.Cert)
	if err != nil {
		return err
	}

	conn, err := tls.Dial("tcp", p.Endpoint, tlsConfig)
	if err != nil {
		return err
	}
	defer conn.Close()

	if _, err := conn.Write([]byte(p.ID)); err != nil {
		return err
	}

	upstrm, err := net.Dial("tcp", "127.0.0.2:32109")
	if err != nil {
		return err
	}
	defer upstrm.Close()

	upstreamConn := connection{
		ReadWriteCloser: upstrm,
		peerKind:        peerServer,
	}
	proxyConn := connection{
		ReadWriteCloser: conn,
		peerKind:        peerClient,
	}

	go func() {
		p.copyWithMeter(ctx, upstreamConn, proxyConn)
		upstreamConn.Close()
		proxyConn.Close()
	}()

	p.copyWithMeter(ctx, proxyConn, upstreamConn)

	return nil
}
