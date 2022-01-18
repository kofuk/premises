package monitor

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"chronoscoper.com/premises/config"
	"github.com/gorilla/websocket"
)

type statusData struct {
	Status   string `json:"status"`
	Shutdown bool   `json:"shutdown"`
	HasError bool   `json:"hasError"`
}

func makeTLSConfig(config *config.Config) (*tls.Config, error) {
	rootCAs := x509.NewCertPool()
	certFile, err := os.ReadFile(filepath.Join(config.Prefix, "/opt/premises/server.crt"))
	if err != nil {
		return nil, err
	}
	rootCAs.AppendCertsFromPEM(certFile)

	return &tls.Config{
		RootCAs: rootCAs,
	}, nil

}

func MonitorServer(cfg *config.Config, addr string, evCh chan string) error {
	tlsConfig, err := makeTLSConfig(cfg)
	if err != nil {
		return err
	}

	dialer := &websocket.Dialer{
		TLSClientConfig: tlsConfig,
	}
	for {
	newConn:
		conn, _, err := dialer.Dial("wss://"+addr+"/monitor", http.Header{"X-Auth-Key": []string{cfg.MonitorKey}})
		if err != nil {
			log.Println(err)
			time.Sleep(10 * time.Second)
			goto newConn
		}
		defer conn.Close()

		for {
			var status statusData
			if err := conn.ReadJSON(&status); err != nil {
				log.Println(err)
				goto newConn
			}

			evCh <- status.Status

			if status.Shutdown {
				close(evCh)
				goto end
			}
		}
	}
end:
	return nil
}

func StopServer(cfg *config.Config, addr string) error {
	tlsConfig, err := makeTLSConfig(cfg)
	client := http.Client{
		Transport: &http.Transport{
			TLSClientConfig: tlsConfig,
		},
	}

	req, err := http.NewRequest("GET", "https://"+addr+"/stop", nil)
	if err != nil {
		return err
	}
	req.Header.Add("X-Auth-Key", cfg.MonitorKey)

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return errors.New(fmt.Sprintf("StopServer: request failed: %d", resp.StatusCode))
	}
	return nil
}
