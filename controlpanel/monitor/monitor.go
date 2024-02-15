package monitor

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"time"

	entityTypes "github.com/kofuk/premises/common/entity"
	runnerEntity "github.com/kofuk/premises/common/entity/runner"
	entity "github.com/kofuk/premises/common/entity/web"
	"github.com/kofuk/premises/controlpanel/config"
	"github.com/kofuk/premises/controlpanel/conoha"
	"github.com/kofuk/premises/controlpanel/dns"
	"github.com/kofuk/premises/controlpanel/kvs"
	"github.com/kofuk/premises/controlpanel/streaming"
)

type StatusData struct {
	Type     string  `json:"type"`
	Status   string  `json:"status"`
	Shutdown bool    `json:"shutdown"`
	HasError bool    `json:"hasError"`
	CPUUsage float64 `json:"cpuUsage"`
}

func GetPageCodeByEventCode(event entityTypes.EventCode) entity.PageCode {
	if event == runnerEntity.EventRunning {
		return entity.PageRunning
	}
	return entity.PageLoading
}

func UpdateRunnerID(ctx context.Context, cfg *config.Config, cache *kvs.KeyValueStore, ipv4Addr string) error {
	var id string
	if err := cache.Get(ctx, "runner-id:default", &id); err == nil {
		return nil
	}

	slog.Debug("Updating runner ID")

	token, _, err := conoha.GetToken(ctx, cfg)
	if err != nil {
		return err
	}

	vm, err := conoha.FindVM(ctx, cfg, token, conoha.FindByIPAddr(ipv4Addr))
	if err != nil {
		return err
	}

	if err := cache.Set(ctx, "runner-id:default", vm.ID, -1); err != nil {
		return err
	}

	return nil
}

func HandleEvent(ctx context.Context, runnerId string, strmProvider *streaming.StreamingService, cfg *config.Config, kvs *kvs.KeyValueStore, dnsService *dns.DNSService, event *runnerEntity.Event) error {
	stdStream := strmProvider.GetStream(streaming.StandardStream)
	infoStream := strmProvider.GetStream(streaming.InfoStream)
	sysstatStream := strmProvider.GetStream(streaming.SysstatStream)

	switch event.Type {
	case runnerEntity.EventHello:
		if event.Hello == nil {
			return errors.New("Invalid event message: has no Hello")
		}
		if err := kvs.Set(ctx, fmt.Sprintf("runner-info:%s", runnerId), event.Hello, 30*24*time.Hour); err != nil {
			return err
		}

		if len(event.Hello.Addr.IPv4) != 0 {
			if err := UpdateRunnerID(ctx, cfg, kvs, event.Hello.Addr.IPv4[0]); err != nil {
				slog.Error("Error updating runner ID", slog.Any("error", err))
			}

			if dnsService != nil {
				if err := dnsService.UpdateV4(ctx, net.ParseIP(event.Hello.Addr.IPv4[0])); err != nil {
					slog.Error("Failed to update IPv4 address", slog.Any("error", err))

					if err := strmProvider.PublishEvent(
						ctx,
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
			ctx,
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
			ctx,
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
			ctx,
			infoStream,
			streaming.NewInfoMessage(event.Info.InfoCode, event.Info.IsError),
		); err != nil {
			return err
		}

	case runnerEntity.EventStarted:
		if event.Started == nil {
			return errors.New("Invalid event message: has no Started")
		}

		if err := kvs.Set(ctx, fmt.Sprintf("world-info:%s", runnerId), event.Started, 30*24*time.Hour); err != nil {
			return err
		}
	}
	return nil
}

func GetSystemInfo(ctx context.Context, cfg *config.Config, cache *kvs.KeyValueStore) (*entity.SystemInfo, error) {
	var serverHello runnerEntity.HelloExtra
	if err := cache.Get(ctx, "runner-info:default", &serverHello); err != nil {
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

func GetWorldInfo(ctx context.Context, cfg *config.Config, cache *kvs.KeyValueStore) (*entity.WorldInfo, error) {
	var startedData runnerEntity.StartedExtra
	if err := cache.Get(ctx, "world-info:default", &startedData); err != nil {
		return nil, err
	}

	return &entity.WorldInfo{
		Version:   startedData.ServerVersion,
		WorldName: startedData.World.Name,
		Seed:      startedData.World.Seed,
	}, nil
}
