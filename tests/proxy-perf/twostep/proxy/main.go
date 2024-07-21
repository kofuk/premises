package main

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"io"
	"log"
	"math/big"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
)

var (
	connMap = make(map[string]chan net.Conn)
)

func handleRequest(conn net.Conn) {
	defer conn.Close()

	id := uuid.New().String()
	connMap[id] = make(chan net.Conn)

	resp, err := http.Post(os.Getenv("UPSTREAM_API"), "text/plain", strings.NewReader(id))
	if err != nil {
		log.Printf("Error connecting to upstream: %v", err)
		return
	}
	io.Copy(io.Discard, resp.Body)

	upstream := <-connMap[id]
	defer upstream.Close()

	go func() {
		_, err := io.Copy(upstream, conn)
		if err != nil {
			log.Printf("Error copying data to upstream: %v", err)
		}
		conn.Close()
		upstream.Close()
	}()

	_, err = io.Copy(conn, upstream)
	if err != nil {
		log.Printf("Error copying data to client: %v", err)
	}
}

func startProxy() error {
	l, err := net.Listen("tcp", os.Getenv("PROXY_LISTEN_ADDR"))
	if err != nil {
		return err
	}
	defer l.Close()

	for {
		conn, err := l.Accept()
		if err != nil {
			return err
		}
		go handleRequest(conn)
	}
}

func waitConnFromUpstream() {
	keyPair, err := tls.LoadX509KeyPair("cert.pem", "key.pem")
	if err != nil {
		log.Fatalf("Error loading key pair: %v", err)
		return
	}

	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{keyPair},
	}
	l, err := tls.Listen("tcp", os.Getenv("UPSTREAM_LISTEN_ADDR"), tlsConfig)
	if err != nil {
		log.Fatalf("Error listening upstream: %v", err)
		return
	}
	defer l.Close()

	for {
		conn, err := l.Accept()
		if err != nil {
			log.Printf("Error accepting upstream connection: %v", err)
			continue
		}
		buf := make([]byte, 36)
		if _, err := io.ReadFull(conn, buf); err != nil {
			log.Printf("Error reading from upstream: %v", err)
			conn.Close()
			continue
		}

		connMap[string(buf)] <- conn
	}
}

func generateCertificate() error {
	priv, err := rsa.GenerateKey(rand.Reader, 4096)
	if err != nil {
		return err
	}
	pub := priv.Public()
	template := x509.Certificate{
		SerialNumber: big.NewInt(1),
		NotAfter:     time.Date(time.Now().Year()+100, 1, 1, 0, 0, 0, 0, time.UTC),
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		DNSNames:     []string{"test.example.com"},
	}
	cert, err := x509.CreateCertificate(rand.Reader, &template, &template, pub, priv)
	if err != nil {
		return err
	}

	var certPem, keyPem strings.Builder
	if err := pem.Encode(&certPem, &pem.Block{Type: "CERTIFICATE", Bytes: cert}); err != nil {
		return err
	}

	if err := pem.Encode(&keyPem, &pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(priv)}); err != nil {
		return err
	}

	if err := os.WriteFile("cert.pem", []byte(certPem.String()), 0644); err != nil {
		return err
	}
	if err := os.WriteFile("key.pem", []byte(keyPem.String()), 0644); err != nil {
		return err
	}

	return nil
}

func main() {
	if err := generateCertificate(); err != nil {
		log.Fatalf("Error generating certificate: %v", err)
	}

	go waitConnFromUpstream()
	startProxy()
}
