package monitor

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/go-redis/redis/v8"
	entityTypes "github.com/kofuk/premises/common/entity"
	runnerEntity "github.com/kofuk/premises/common/entity/runner"
	entity "github.com/kofuk/premises/common/entity/web"
	"github.com/kofuk/premises/controlpanel/config"
	"github.com/kofuk/premises/controlpanel/streaming"
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

func HandleEvent(strmProvider *streaming.Streaming, cfg *config.Config, rdb *redis.Client, event *runnerEntity.Event) error {
	stdStream := strmProvider.GetStream(streaming.StandardStream)
	infoStream := strmProvider.GetStream(streaming.InfoStream)
	sysstatStream := strmProvider.GetStream(streaming.SysstatStream)

	if event.Type == runnerEntity.EventStatus {
		if event.Status == nil {
			return errors.New("Invalid event message: has no Status")
		}

		if err := strmProvider.PublishEvent(
			context.Background(),
			stdStream,
			streaming.NewStandardMessageWithProgress(event.Status.EventCode, event.Status.Progress, GetPageCodeByEventCode(event.Status.EventCode)),
		); err != nil {
			return err
		}
	} else if event.Type == runnerEntity.EventSysstat {
		if event.Sysstat == nil {
			return errors.New("Invalid event message: has no Sysstat")
		}

		if err := strmProvider.PublishEvent(
			context.Background(),
			sysstatStream,
			streaming.NewSysstatMessage(event.Sysstat.CPUUsage, event.Sysstat.Time),
		); err != nil {
			return err
		}
	} else if event.Type == runnerEntity.EventInfo {
		if event.Info == nil {
			return errors.New("Invalid event message: has no Info")
		}

		if err := strmProvider.PublishEvent(
			context.Background(),
			infoStream,
			streaming.NewInfoMessage(event.Info.InfoCode, event.Info.IsError),
		); err != nil {
			return err
		}
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
