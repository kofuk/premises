package monitor

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"time"

	"github.com/go-redis/redis/v8"
	entityTypes "github.com/kofuk/premises/common/entity"
	runnerEntity "github.com/kofuk/premises/common/entity/runner"
	entity "github.com/kofuk/premises/common/entity/web"
	"github.com/kofuk/premises/controlpanel/caching"
	"github.com/kofuk/premises/controlpanel/config"
	"github.com/kofuk/premises/controlpanel/conoha"
	"github.com/kofuk/premises/controlpanel/dns"
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

func UpdateRunnerID(cfg *config.Config, cache *caching.Cacher, ipv4Addr string) error {
	var id string
	if err := cache.Get(context.Background(), "runner-id:default", &id); err == nil {
		return nil
	}

	slog.Debug("Updating runner ID")

	token, _, err := conoha.GetToken(cfg)
	if err != nil {
		return err
	}

	vm, err := conoha.FindVM(cfg, token, conoha.FindByIPAddr(ipv4Addr))
	if err != nil {
		return err
	}

	if err := cache.Set(context.Background(), "runner-id:default", vm.ID, -1); err != nil {
		return err
	}

	return nil
}

func HandleEvent(runnerId string, strmProvider *streaming.Streaming, cfg *config.Config, cache *caching.Cacher, dnsService *dns.DNSProvider, event *runnerEntity.Event) error {
	stdStream := strmProvider.GetStream(streaming.StandardStream)
	infoStream := strmProvider.GetStream(streaming.InfoStream)
	sysstatStream := strmProvider.GetStream(streaming.SysstatStream)

	switch event.Type {
	case runnerEntity.EventHello:
		if event.Hello == nil {
			return errors.New("Invalid event message: has no Hello")
		}
		if err := cache.Set(context.Background(), fmt.Sprintf("runner-info:%s", runnerId), event.Hello, 30*24*time.Hour); err != nil {
			return err
		}

		if len(event.Hello.Addr.IPv4) != 0 {
			if err := UpdateRunnerID(cfg, cache, event.Hello.Addr.IPv4[0]); err != nil {
				slog.Error("Error updating runner ID", slog.Any("error", err))
			}

			if dnsService != nil {
				if err := dnsService.UpdateV4(context.Background(), net.ParseIP(event.Hello.Addr.IPv4[0])); err != nil {
					slog.Error("Failed to update IPv4 address", slog.Any("error", err))

					if err := strmProvider.PublishEvent(
						context.Background(),
						infoStream,
						streaming.NewInfoMessage(entity.InfoErrDNS, true),
					); err != nil {
						slog.Error("Failed to write status data to Redis channel", slog.Any("error", err))
					}
				}
			}
		}

	case runnerEntity.EventStatus:
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

	case runnerEntity.EventSysstat:
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

	case runnerEntity.EventInfo:
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

	case runnerEntity.EventStarted:
		if event.Started == nil {
			return errors.New("Invalid event message: has no Started")
		}

		if err := cache.Set(context.Background(), fmt.Sprintf("world-info:%s", runnerId), event.Started, 30*24*time.Hour); err != nil {
			return err
		}
	}
	return nil
}

func GetSystemInfo(ctx context.Context, cfg *config.Config, cache *caching.Cacher) (*entity.SystemInfo, error) {
	var serverHello runnerEntity.HelloExtra
	if err := cache.Get(context.Background(), "runner-info:default", &serverHello); err != nil {
		return nil, err
	}

	var ipAddr *string
	if len(serverHello.Addr.IPv4) != 0 {
		ipAddr = &serverHello.Addr.IPv4[0]
	}

	return &entity.SystemInfo{
		PremisesVersion: serverHello.Version,
		HostOS:          serverHello.Host,
		IPAddress:       ipAddr,
	}, nil
}

func GetWorldInfo(ctx context.Context, cfg *config.Config, cache *caching.Cacher) (*entity.WorldInfo, error) {
	var startedData runnerEntity.StartedExtra
	if err := cache.Get(context.Background(), "world-info:default", &startedData); err != nil {
		return nil, err
	}

	return &entity.WorldInfo{
		Version:   startedData.ServerVersion,
		WorldName: startedData.World.Name,
		Seed:      startedData.World.Seed,
	}, nil
}
