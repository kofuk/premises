package monitor

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"math/big"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/kofuk/premises/controlpanel/config"
)

func GenerateTLSKey(cfg *config.Config, rdb *redis.Client) error {
	privKey, err := rsa.GenerateKey(rand.Reader, 4096)
	if err != nil {
		return err
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

	pubKeyBuf := new(bytes.Buffer)
	if err := pem.Encode(pubKeyBuf, &pem.Block{Type: "CERTIFICATE", Bytes: cert}); err != nil {
		return err
	}

	privKeyX509 := x509.MarshalPKCS1PrivateKey(privKey)
	privKeyBuf := new(bytes.Buffer)
	if err := pem.Encode(privKeyBuf, &pem.Block{Type: "RSA PRIVATE KEY", Bytes: privKeyX509}); err != nil {
		return err
	}

	if _, err := rdb.Set(context.Background(), "server-key", privKeyBuf.String(), 0).Result(); err != nil {
		return err
	}
	if _, err := rdb.Set(context.Background(), "server-crt", pubKeyBuf.String(), 0).Result(); err != nil {
		return err
	}

	return nil
}
