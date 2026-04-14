package proxy

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"math/big"
	"strings"
	"time"
)

type Certificate struct {
	Cert, Key string
}

func generateCertificate() (*Certificate, error) {
	priv, err := rsa.GenerateKey(rand.Reader, 4096)
	if err != nil {
		return nil, err
	}
	pub := priv.Public()
	template := x509.Certificate{
		SerialNumber: big.NewInt(1),
		NotAfter:     time.Date(time.Now().Year()+100, 1, 1, 0, 0, 0, 0, time.UTC),
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		DNSNames:     []string{"control.fake.premises.kofuk.org"},
	}
	cert, err := x509.CreateCertificate(rand.Reader, &template, &template, pub, priv)
	if err != nil {
		return nil, err
	}

	var certPem, keyPem strings.Builder
	if err := pem.Encode(&certPem, &pem.Block{Type: "CERTIFICATE", Bytes: cert}); err != nil {
		return nil, err
	}

	if err := pem.Encode(&keyPem, &pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(priv)}); err != nil {
		return nil, err
	}

	return &Certificate{Cert: certPem.String(), Key: keyPem.String()}, nil
}
