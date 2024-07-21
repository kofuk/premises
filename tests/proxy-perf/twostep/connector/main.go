package main

import (
	"crypto/tls"
	"crypto/x509"
	"io"
	"log"
	"net"
	"net/http"
	"os"
)

func openConnection(id string) {
	cert, err := os.ReadFile("cert.pem")
	if err != nil {
		log.Printf("Error reading certificate: %v", err)
		return
	}

	rootCAs := x509.NewCertPool()
	if ok := rootCAs.AppendCertsFromPEM(cert); !ok {
		log.Println("Error appending certificate")
		return
	}

	tlsConfig := &tls.Config{
		RootCAs:    rootCAs,
		ServerName: "test.example.com",
	}

	conn, err := tls.Dial("tcp", os.Getenv("DOWNSTREAM_ADDR"), tlsConfig)
	if err != nil {
		log.Printf("Error connecting to downstream: %v", err)
		return
	}
	defer conn.Close()

	if _, err := conn.Write([]byte(id)); err != nil {
		log.Printf("Error writing connection ID to downstream: %v", err)
		return
	}

	upstream, err := net.Dial("tcp", os.Getenv("CONNECTOR_UPSTREAM_ADDR"))
	if err != nil {
		log.Printf("Error connecting to upstream: %v", err)
		return
	}
	defer upstream.Close()

	go func() {
		_, err := io.Copy(upstream, conn)
		if err != nil {
			log.Printf("Error copying data to upstream: %v", err)
		}
		conn.Close()
		upstream.Close()
	}()

	if _, err = io.Copy(conn, upstream); err != nil {
		log.Printf("Error copying data to client: %v", err)
	}
}

func main() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		d, err := io.ReadAll(r.Body)
		if err != nil {
			log.Println("Error reading request body:", err)
			return
		}

		go openConnection(string(d))
	})
	http.ListenAndServe(os.Getenv("API_LISTEN_ADDR"), http.DefaultServeMux)
}
