package monitor

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"math/big"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/kofuk/premises/controlpanel/config"
)

func generateCertPem() (certPem, keyPem []byte, err error) {
	privKey, err := rsa.GenerateKey(rand.Reader, 4096)
	if err != nil {
		return nil, nil, err
	}
	pubKey := privKey.Public()
	caTemplate := x509.Certificate{
		SerialNumber: big.NewInt(1),
		NotAfter:     time.Date(time.Now().Year()+100, 1, 1, 0, 0, 0, 0, time.UTC),
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		DNSNames:     []string{"usergameservermonitoring.premises.kofuk.org"},
	}
	cert, err := x509.CreateCertificate(rand.Reader, &caTemplate, &caTemplate, pubKey, privKey)

	certBuf := new(bytes.Buffer)
	if err := pem.Encode(certBuf, &pem.Block{Type: "CERTIFICATE", Bytes: cert}); err != nil {
		return nil, nil, err
	}

	privKeyX509 := x509.MarshalPKCS1PrivateKey(privKey)
	privKeyBuf := new(bytes.Buffer)
	if err := pem.Encode(privKeyBuf, &pem.Block{Type: "RSA PRIVATE KEY", Bytes: privKeyX509}); err != nil {
		return nil, nil, err
	}

	return certBuf.Bytes(), privKeyBuf.Bytes(), nil
}

func GenerateTLSKey(cfg *config.Config, rdb *redis.Client) error {
	privKey, pubKey, err := generateCertPem()
	if err != nil {
		return err
	}

	if _, err := rdb.Set(context.Background(), "server-key", privKey, 0).Result(); err != nil {
		return err
	}
	if _, err := rdb.Set(context.Background(), "server-crt", pubKey, 0).Result(); err != nil {
		return err
	}

	return nil
}

func makeTLSClientConfigWithCert(cert []byte) (*tls.Config, error) {
	rootCAs := x509.NewCertPool()

	rootCAs.AppendCertsFromPEM(cert)

	return &tls.Config{
		RootCAs:    rootCAs,
		ServerName: "usergameservermonitoring.premises.kofuk.org",
	}, nil
}

func makeTLSClientConfig(config *config.Config, rdb *redis.Client) (*tls.Config, error) {
	certFile, err := rdb.Get(context.Background(), "server-crt").Result()
	if err != nil {
		return nil, err
	}
	return makeTLSClientConfigWithCert([]byte(certFile))
}
