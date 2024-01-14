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
	entityTypes "github.com/kofuk/premises/common/entity"
	runnerEntity "github.com/kofuk/premises/common/entity/runner"
	entity "github.com/kofuk/premises/common/entity/web"
	"github.com/kofuk/premises/controlpanel/config"
	"github.com/kofuk/premises/controlpanel/streaming"
	log "github.com/sirupsen/logrus"
)

type StatusData struct {
	Type     string  `json:"type"`
	Status   string  `json:"status"`
	Shutdown bool    `json:"shutdown"`
	HasError bool    `json:"hasError"`
	CPUUsage float64 `json:"cpuUsage"`
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

func GetPageCodeByEventCode(event entityTypes.EventCode) entity.PageCode {
	if event == runnerEntity.EventRunning {
		return entity.PageRunning
	}
	return entity.PageLoading
}

func MonitorServer(strmProvider *streaming.Streaming, cfg *config.Config, addr string, rdb *redis.Client) error {
	tlsConfig, err := makeTLSClientConfig(cfg, rdb)
	if err != nil {
		return err
	}

	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.TLSClientConfig = tlsConfig
	client := http.Client{
		Transport: transport,
	}

	stdStream := strmProvider.GetStream(streaming.StandardStream)
	infoStream := strmProvider.GetStream(streaming.InfoStream)
	sysstatStream := strmProvider.GetStream(streaming.SysstatStream)

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
				if err := strmProvider.PublishEvent(
					context.Background(),
					stdStream,
					streaming.NewStandardMessage(entity.EvConnLost, entity.PageLoading),
				); err != nil {
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

		receiveEvent := func(reader *bufio.Reader) (*runnerEntity.Event, error) {
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

			var event runnerEntity.Event
			if err := json.Unmarshal(line, &event); err != nil {
				return nil, err
			}

			return &event, nil
		}

		respReader := bufio.NewReader(resp.Body)

	conn:
		for {
			event, err := receiveEvent(respReader)
			if err != nil {
				log.WithError(err).Error("Failed to receive event data")
				break conn
			}

			if event.Type == runnerEntity.EventStatus {
				if event.Status == nil {
					log.WithField("event", event).Error("Invalid event message (has no Status)")
					continue
				}

				if event.Status.EventCode == runnerEntity.EventShutdown {
					break out
				}

				if err := strmProvider.PublishEvent(
					context.Background(),
					stdStream,
					streaming.NewStandardMessageWithProgress(event.Status.EventCode, event.Status.Progress, GetPageCodeByEventCode(event.Status.EventCode)),
				); err != nil {
					log.WithError(err).Error("Failed to write status data to Redis channel")
				}
			} else if event.Type == runnerEntity.EventSysstat {
				if event.Sysstat == nil {
					log.WithField("event", event).Error("Invalid event message (has no Sysstat)")
					continue
				}

				if err := strmProvider.PublishEvent(
					context.Background(),
					sysstatStream,
					streaming.NewSysstatMessage(event.Sysstat.CPUUsage, event.Sysstat.Time),
				); err != nil {
					log.WithError(err).Error("Failed to write status data to Redis channel")
				}
			} else if event.Type == runnerEntity.EventInfo {
				if event.Info == nil {
					log.WithField("event", event).Error("Invalid event message (has no Info)")
					continue
				}

				if err := strmProvider.PublishEvent(
					context.Background(),
					infoStream,
					streaming.NewInfoMessage(event.Info.InfoCode, event.Info.IsError),
				); err != nil {
					log.WithError(err).Error("Failed to write status data to Redis channel")
				}
			}
		}

		resp.Body.Close()

		connLost = true

		time.Sleep(2 * time.Second)
		startTime = time.Now()
	}

	// The server is about to shutdown, we no longer need sysstat history.
	if err := strmProvider.ClearHistory(context.Background(), sysstatStream); err != nil {
		log.WithError(err).Error("Unable to clear sysstat history")
	}

	return nil

err:
	log.Error("Server did not respond in 10 minutes. Shutting down...")

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

func ReconfigureServer(gameConfig *runnerEntity.Config, cfg *config.Config, addr string, rdb *redis.Client) error {
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

	if resp.StatusCode != http.StatusOK {
		return errors.New("Error processing quick undo")
	}

	return nil
}
