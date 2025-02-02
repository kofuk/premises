package connector

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"io"
	"net"
)

type Proxy struct {
	ID       string
	Endpoint string
	Cert     string
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

func (p *Proxy) Run() error {
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

	upstrm, err := net.Dial("tcp", "localhost:32109")
	if err != nil {
		return err
	}
	defer upstrm.Close()

	go func() {
		io.Copy(upstrm, conn)
		upstrm.Close()
		conn.Close()
	}()

	io.Copy(conn, upstrm)

	return nil
}
