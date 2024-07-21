package main

import (
	"io"
	"log"
	"net"
	"os"
)

func handleRequest(conn net.Conn) {
	defer conn.Close()

	upstream, err := net.Dial("tcp", os.Getenv("UPSTREAM_ADDR"))
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

func startProxy() error {
	l, err := net.Listen("tcp", os.Getenv("LISTEN_ADDR"))
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

func main() {
	if err := startProxy(); err != nil {
		log.Fatalf("Error starting proxy: %v", err)
	}
}
