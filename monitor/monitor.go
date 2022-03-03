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

	"github.com/gorilla/websocket"
	"github.com/kofuk/premises/config"
)

type StatusData struct {
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
		//TODO: Can't we use TLS without setting InsecureSkipVerify???
		InsecureSkipVerify: true,
	}, nil

}

func MonitorServer(cfg *config.Config, addr string, evCh chan *StatusData) error {
	tlsConfig, err := makeTLSConfig(cfg)
	if err != nil {
		return err
	}

	dialer := &websocket.Dialer{
		TLSClientConfig: tlsConfig,
	}
	startTime := time.Now()
	for {
	newConn:
		conn, _, err := dialer.Dial("wss://"+addr+":8521/monitor", http.Header{"X-Auth-Key": []string{cfg.MonitorKey}})
		if err != nil {
			log.Println(err)

			if time.Now().Sub(startTime) > 10*time.Minute {
				goto err
			}

			time.Sleep(10 * time.Second)
			goto newConn
		}
		defer conn.Close()

		for {
			var status StatusData
			if err := conn.ReadJSON(&status); err != nil {
				log.Println(err)

				evCh <- &StatusData{
					Status:   "Connection lost. Will reconnect...",
					HasError: true,
					Shutdown: false,
				}

				startTime = time.Now()
				goto newConn
			}

			evCh <- &status

			if status.Shutdown {
				goto end
			}
		}
	}
end:

	return nil

err:
	evCh <- &StatusData{
		Status:   "Server did not respond in 10 minutes. I'm tired of waiting :P",
		HasError: true,
		Shutdown: false,
	}

	return nil
}

func StopServer(cfg *config.Config, addr string) error {
	tlsConfig, err := makeTLSConfig(cfg)
	client := http.Client{
		Transport: &http.Transport{
			TLSClientConfig: tlsConfig,
		},
	}

	req, err := http.NewRequest("GET", "https://"+addr+":8521/stop", nil)
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
