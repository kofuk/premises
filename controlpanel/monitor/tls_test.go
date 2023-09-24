package monitor

import (
	"crypto/tls"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func runTLSServerWithCert(t *testing.T, cert, key []byte, sockPath string) error {
	keyPair, err := tls.X509KeyPair(cert, key)
	if err != nil {
		return err
	}

	tlsConfig := &tls.Config{
		Certificates:             []tls.Certificate{keyPair},
		MinVersion:               tls.VersionTLS13,
		CurvePreferences:         []tls.CurveID{tls.CurveP521, tls.CurveP384, tls.CurveP256},
		PreferServerCipherSuites: true,
		CipherSuites: []uint16{
			tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA,
			tls.TLS_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_RSA_WITH_AES_256_CBC_SHA,
		},
	}

	listener, err := tls.Listen("unix", sockPath, tlsConfig)
	if err != nil {
		return err
	}
	defer listener.Close()

	conn, err := listener.Accept()
	if err != nil {
		return err
	}

	if _, err := io.ReadAll(conn); err != nil {
		return err
	}

	conn.Close()

	return nil
}

func Test_generateCertPem_shouldSuccess(t *testing.T) {
	cert, key, err := generateCertPem()
	if err != nil {
		t.Fatal(err)
	}

	sockPath := filepath.Join(os.TempDir(), fmt.Sprintf("premises_test_%d.sock", rand.Int()))
	go func() {
		if err := runTLSServerWithCert(t, cert, key, sockPath); err != nil {
			panic(err)
		}
	}()

	for {
		_, err := os.Stat(sockPath)
		if err == nil {
			break
		}

		if os.IsNotExist(err) {
			time.Sleep(5 * time.Millisecond)
			continue
		}

		t.Fatal(err)
	}

	tlsConfig, err := makeTLSClientConfigWithCert(cert)
	if err != nil {
		t.Fatal(err)
	}

	conn, err := tls.Dial("unix", sockPath, tlsConfig)
	if err != nil {
		t.Fatal(err)
	}
	conn.Close()

	assert.True(t, true)
}

func Test_generateCertPem_shouldFailWithUnknownAuthority(t *testing.T) {
	cert, key, err := generateCertPem()
	if err != nil {
		t.Fatal(err)
	}

	sockPath := filepath.Join(os.TempDir(), fmt.Sprintf("premises_test_%d.sock", rand.Int()))
	go func() {
		if err := runTLSServerWithCert(t, cert, key, sockPath); err != nil {
			fmt.Fprintf(os.Stderr, "Error in server: %v", err)
		}
	}()

	for {
		_, err := os.Stat(sockPath)
		if err == nil {
			break
		}

		if os.IsNotExist(err) {
			time.Sleep(5 * time.Millisecond)
			continue
		}

		t.Fatal(err)
	}

	tlsConfig := http.DefaultTransport.(*http.Transport).Clone().TLSClientConfig
	tlsConfig.ServerName = "usergameservermonitoring.premises.kofuk.org"
	conn, err := tls.Dial("unix", sockPath, tlsConfig)
	if err != nil {
		assert.Contains(t, err.Error(), "tls: failed to verify certificate: x509: certificate signed by unknown authority")
	} else {
		conn.Close()
		assert.Fail(t, "Certificate verification should fail")
	}
}

func Test_generateCertPem_shouldFailWithServerNameUnmatch(t *testing.T) {
	cert, key, err := generateCertPem()
	if err != nil {
		t.Fatal(err)
	}

	sockPath := filepath.Join(os.TempDir(), fmt.Sprintf("premises_test_%d.sock", rand.Int()))
	go func() {
		if err := runTLSServerWithCert(t, cert, key, sockPath); err != nil {
			fmt.Fprintf(os.Stderr, "Error in server: %v", err)
		}
	}()

	for {
		_, err := os.Stat(sockPath)
		if err == nil {
			break
		}

		if os.IsNotExist(err) {
			time.Sleep(5 * time.Millisecond)
			continue
		}

		t.Fatal(err)
	}

	tlsConfig := http.DefaultTransport.(*http.Transport).Clone().TLSClientConfig
	conn, err := tls.Dial("unix", sockPath, tlsConfig)
	if err != nil {
		assert.Contains(t, err.Error(), "tls: failed to verify certificate: x509: certificate is valid for usergameservermonitoring.premises.kofuk.org, not ")
	} else {
		conn.Close()
		assert.Fail(t, "Certificate verification should fail")
	}
}
