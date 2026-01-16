package db

import (
	"crypto/tls"
	"crypto/x509"
	"database/sql"
	"os"
	"strings"

	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
	"github.com/uptrace/bun/driver/pgdriver"
)

type ConnOptions struct {
	Addr       string
	User       string
	Password   string
	Database   string
	SSLMode    string
	CACertPath string
}

func loadCertPool(caCertPath string) (*x509.CertPool, error) {
	bytes, err := os.ReadFile(caCertPath)
	if err != nil {
		return nil, err
	}
	certPool := x509.NewCertPool()
	if ok := certPool.AppendCertsFromPEM(bytes); !ok {
		return nil, err
	}
	return certPool, nil
}

func NewClient(options ConnOptions) (*bun.DB, error) {
	opts := []pgdriver.Option{
		pgdriver.WithAddr(options.Addr),
		pgdriver.WithUser(options.User),
		pgdriver.WithPassword(options.Password),
		pgdriver.WithDatabase(options.Database),
		pgdriver.WithConnParams(map[string]any{
			"TimeZone": "Etc/UTC",
		}),
	}

	if options.SSLMode == "verify-full" {
		parts := strings.SplitN(options.Addr, ":", 2)
		tlsConfig := &tls.Config{
			ServerName: parts[0],
		}
		if options.CACertPath != "" {
			certPool, err := loadCertPool(options.CACertPath)
			if err != nil {
				return nil, err
			}
			tlsConfig.RootCAs = certPool
		}

		opts = append(opts, pgdriver.WithTLSConfig(tlsConfig))
	} else {
		opts = append(opts, pgdriver.WithInsecure(true))
	}

	conn := pgdriver.NewConnector(
		opts...,
	)
	return bun.NewDB(sql.OpenDB(conn), pgdialect.New()), nil
}
