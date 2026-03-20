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

func copyWithMeter(ctx context.Context, dst io.Writer, src io.Reader, counter metric.Int64Counter, peer string) {
	buf := make([]byte, 32*1024)
	for {
		n, err := src.Read(buf)
		if n > 0 {
			if _, err := dst.Write(buf[:n]); err != nil {
				return
			}
			counter.Add(ctx, int64(n), metric.WithAttributes(
				attribute.String("peer", peer),
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

	go func() {
		copyWithMeter(ctx, upstrm, conn, p.Metrics.io, "server")
		upstrm.Close()
		conn.Close()
	}()

	copyWithMeter(ctx, conn, upstrm, p.Metrics.io, "client")

	return nil
}
