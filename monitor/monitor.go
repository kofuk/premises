package monitor

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/gorilla/websocket"
	"github.com/kofuk/premises/config"
	"github.com/kofuk/premises/gameconfig"
	log "github.com/sirupsen/logrus"
)

type StatusData struct {
	Status   string `json:"status"`
	Shutdown bool   `json:"shutdown"`
	HasError bool   `json:"hasError"`
}

func makeTLSConfig(config *config.Config) (*tls.Config, error) {
	rootCAs := x509.NewCertPool()
	certFile, err := os.ReadFile(config.Locate("server.crt"))
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
	connLost := false
	startTime := time.Now()
	for {
	newConn:
		conn, _, err := dialer.Dial("wss://"+addr+":8521/monitor", http.Header{"X-Auth-Key": []string{cfg.MonitorKey}})
		if err != nil {
			log.WithError(err).Error("Failed to connect to status server")

			if connLost {
				evCh <- &StatusData{
					Status:   "Connection lost. Will reconnect...",
					HasError: true,
					Shutdown: false,
				}

				connLost = false
			}

			if time.Now().Sub(startTime) > 10*time.Minute {
				goto err
			}

			time.Sleep(10 * time.Second)
			goto newConn
		}
		defer conn.Close()

		connLost = false

		for {
			var status StatusData
			if err := conn.ReadJSON(&status); err != nil {
				log.WithError(err).Error("Failed to read data")

				connLost = true

				time.Sleep(2 * time.Second)

				startTime = time.Now()
				goto newConn
			}

			// Don't send "shutdown" event.
			// We'll send one to clients after cleaning up VMs.
			if status.Shutdown {
				goto end
			}

			evCh <- &status
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

func ReconfigureServer(gameConfig *gameconfig.GameConfig, cfg *config.Config, addr string) error {
	tlsConfig, err := makeTLSConfig(cfg)
	client := http.Client{
		Transport: &http.Transport{
			TLSClientConfig: tlsConfig,
		},
	}

	data, err := json.Marshal(gameConfig)
	if err != nil {
		return err
	}

	buf := bytes.NewBuffer(data)

	req, err := http.NewRequest("POST", "https://"+addr+":8521/newconfig", buf)
	if err != nil {
		return err
	}
	req.Header.Add("X-Auth-Key", cfg.MonitorKey)

	resp, err := client.Do(req)
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK {
		return errors.New(fmt.Sprintf("Request failed with %d", resp.StatusCode))
	}

	return nil
}

func GetSystemInfoData(cfg *config.Config, addr string) ([]byte, error) {
	tlsConfig, err := makeTLSConfig(cfg)
	client := http.Client{
		Transport: &http.Transport{
			TLSClientConfig: tlsConfig,
		},
	}

	req, err := http.NewRequest("POST", "https://"+addr+":8521/systeminfo", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Add("X-Auth-Key", cfg.MonitorKey)

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return data, nil
}

func TakeSnapshot(cfg *config.Config, addr string) error {
	tlsConfig, err := makeTLSConfig(cfg)
	client := http.Client{
		Transport: &http.Transport{
			TLSClientConfig: tlsConfig,
		},
	}

	req, err := http.NewRequest("POST", "https://"+addr+":8521/snapshot", nil)
	if err != nil {
		return err
	}
	req.Header.Add("X-Auth-Key", cfg.MonitorKey)

	resp, err := client.Do(req)
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusCreated {
		return errors.New("Error creating snapshot")
	}

	return nil
}
