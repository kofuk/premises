package monitor

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/kofuk/premises/controlpanel/config"
	"github.com/kofuk/premises/controlpanel/gameconfig"
	log "github.com/sirupsen/logrus"
)

type StatusData struct {
	Type     string  `json:"type"`
	Status   string  `json:"status"`
	Shutdown bool    `json:"shutdown"`
	HasError bool    `json:"hasError"`
	CPUUsage float64 `json:"cpuUsage"`
}

func PublishEvent(rdb *redis.Client, status StatusData) error {
	jsonData, err := json.Marshal(status)
	if err != nil {
		return err
	}

	if _, err := rdb.Pipelined(context.Background(), func(p redis.Pipeliner) error {
		p.Set(context.Background(), "last-status:default", jsonData, -1)
		p.Publish(context.Background(), "status:default", string(jsonData))
		return nil
	}); err != nil {
		return err
	}

	return nil
}

func publishSystemStatEvent(redis *redis.Client, status StatusData) error {
	json, err := json.Marshal(status)
	if err != nil {
		return err
	}
	if err := redis.Publish(context.TODO(), "systemstat:default", string(json)).Err(); err != nil {
		return err
	}

	return nil
}

func MonitorServer(cfg *config.Config, addr string, rdb *redis.Client) error {
	tlsConfig, err := makeTLSClientConfig(cfg, rdb)
	if err != nil {
		return err
	}

	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.TLSClientConfig = tlsConfig
	client := http.Client{
		Transport: transport,
	}

	connLost := false
	startTime := time.Now()
out:
	for {
		req, err := http.NewRequest(http.MethodGet, "https://"+addr+":8521/monitor", nil)
		if err != nil {
			return err
		}
		req.Header.Add("X-Auth-Key", cfg.MonitorKey)

		resp, err := client.Do(req)
		if err != nil {
			log.WithError(err).Error("Failed to connect to status server")

			if connLost {
				if err := PublishEvent(rdb, StatusData{
					Status:   "Connection lost. Will reconnect...",
					HasError: true,
					Shutdown: false,
				}); err != nil {
					log.WithError(err).Error("Failed to write status data to Redis channel")
				}

				connLost = false
			}

			if time.Now().Sub(startTime) > 10*time.Minute {
				goto err
			}

			time.Sleep(5 * time.Second)
			continue
		}

		connLost = false

		receiveEvent := func(reader *bufio.Reader) (*StatusData, error) {
			var line []byte

			for {
				var err error
				line, _, err = reader.ReadLine()
				if err != nil {
					return nil, err
				}
				if len(line) > 0 && line[0] == ':' {
					continue
				}

				break
			}

			var status StatusData
			if err := json.Unmarshal(line, &status); err != nil {
				return nil, err
			}

			return &status, nil
		}

		respReader := bufio.NewReader(resp.Body)

	conn:
		for {
			status, err := receiveEvent(respReader)
			if err != nil {
				log.WithError(err).Error("Failed to receive event data")
				break conn
			}
			if status.Shutdown {
				resp.Body.Close()
				break out
			}

			// TODO: Remove Type=="" case later
			if status.Type == "" || status.Type == "legacyEvent" {
				if err := PublishEvent(rdb, *status); err != nil {
					log.WithError(err).Error("Failed to write status data to Redis channel")
				}
			} else if status.Type == "systemStat" {
				if err := publishSystemStatEvent(rdb, *status); err != nil {
					log.WithError(err).Error("Failed to write system stat to Redis channel")
				}
			}
		}

		resp.Body.Close()

		connLost = true

		time.Sleep(2 * time.Second)
		startTime = time.Now()
	}

	return nil

err:
	if err := PublishEvent(rdb, StatusData{
		Status:   "Server did not respond in 10 minutes. I'm tired of waiting :P",
		HasError: true,
		Shutdown: false,
	}); err != nil {
		log.WithError(err).Error("Failed to write status data to Redis channel")
	}

	return nil
}

func StopServer(cfg *config.Config, addr string, rdb *redis.Client) error {
	tlsConfig, err := makeTLSClientConfig(cfg, rdb)
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

func ReconfigureServer(gameConfig *gameconfig.GameConfig, cfg *config.Config, addr string, rdb *redis.Client) error {
	tlsConfig, err := makeTLSClientConfig(cfg, rdb)
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

func GetSystemInfoData(ctx context.Context, cfg *config.Config, addr string, rdb *redis.Client) ([]byte, error) {
	tlsConfig, err := makeTLSClientConfig(cfg, rdb)
	client := http.Client{
		Transport: &http.Transport{
			TLSClientConfig: tlsConfig,
		},
	}

	req, err := http.NewRequest("POST", "https://"+addr+":8521/systeminfo", nil)
	if err != nil {
		return nil, err
	}
	req = req.WithContext(ctx)
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

func GetWorldInfoData(ctx context.Context, cfg *config.Config, addr string, rdb *redis.Client) ([]byte, error) {
	tlsConfig, err := makeTLSClientConfig(cfg, rdb)
	client := http.Client{
		Transport: &http.Transport{
			TLSClientConfig: tlsConfig,
		},
	}

	req, err := http.NewRequest("POST", "https://"+addr+":8521/worldinfo", nil)
	if err != nil {
		return nil, err
	}
	req = req.WithContext(ctx)
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

func TakeSnapshot(cfg *config.Config, addr string, rdb *redis.Client) error {
	tlsConfig, err := makeTLSClientConfig(cfg, rdb)
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

func QuickSnapshot(cfg *config.Config, addr string, rdb *redis.Client) error {
	tlsConfig, err := makeTLSClientConfig(cfg, rdb)
	client := http.Client{
		Transport: &http.Transport{
			TLSClientConfig: tlsConfig,
		},
	}

	req, err := http.NewRequest("POST", "https://"+addr+":8521/quickss", nil)
	if err != nil {
		return err
	}
	req.Header.Add("X-Auth-Key", cfg.MonitorKey)

	resp, err := client.Do(req)
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusCreated {
		return errors.New("Error creating quick snapshot")
	}

	return nil
}

func QuickUndo(cfg *config.Config, addr string, rdb *redis.Client) error {
	tlsConfig, err := makeTLSClientConfig(cfg, rdb)
	client := http.Client{
		Transport: &http.Transport{
			TLSClientConfig: tlsConfig,
		},
	}

	req, err := http.NewRequest("POST", "https://"+addr+":8521/quickundo", nil)
	if err != nil {
		return err
	}
	req.Header.Add("X-Auth-Key", cfg.MonitorKey)

	resp, err := client.Do(req)
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusCreated {
		return errors.New("Error processing quick undo")
	}

	return nil
}
