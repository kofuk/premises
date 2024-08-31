package monitor

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/kofuk/premises/controlpanel/internal/config"
	"github.com/kofuk/premises/controlpanel/internal/conoha"
	"github.com/kofuk/premises/controlpanel/internal/kvs"
	"github.com/kofuk/premises/controlpanel/internal/streaming"
	"github.com/kofuk/premises/internal/entity"
	"github.com/kofuk/premises/internal/entity/runner"
	"github.com/kofuk/premises/internal/entity/web"
)

type StatusData struct {
	Type     string  `json:"type"`
	Status   string  `json:"status"`
	Shutdown bool    `json:"shutdown"`
	HasError bool    `json:"hasError"`
	CPUUsage float64 `json:"cpuUsage"`
}

func GetPageCodeByEventCode(event entity.EventCode) web.PageCode {
	if event == entity.EventRunning {
		return web.PageRunning
	}
	return web.PageLoading
}

func AttachRunner(ctx context.Context, cfg *config.Config, cache *kvs.KeyValueStore, ipv4Addr string) error {
	if cfg.ConohaUser == "" || cfg.ConohaPassword == "" {
		return nil
	}

	var id string
	if err := cache.Get(ctx, "runner-id:default", &id); err == nil {
		return nil
	}

	slog.Debug("Updating runner ID")

	identity := conoha.Identity{
		User:     cfg.ConohaUser,
		Password: cfg.ConohaPassword,
		TenantID: cfg.ConohaTenantID,
	}
	endpoints := conoha.Endpoints{
		Identity: cfg.ConohaIdentityService,
		Compute:  cfg.ConohaComputeService,
		Image:    cfg.ConohaImageService,
		Volume:   cfg.ConohaVolumeService,
	}
	client := conoha.NewClient(identity, endpoints, nil)

	servers, err := client.ListServerDetails(ctx)
	if err != nil {
		return err
	}

	var matchingServer *conoha.ServerDetail
out:
	for _, server := range servers.Servers {
		for _, addresses := range server.Addresses {
			for _, addr := range addresses {
				if addr.Version != 4 {
					continue
				}
				if addr.Addr == ipv4Addr {
					matchingServer = &server
					break out
				}
			}
		}
	}

	if matchingServer == nil {
		return errors.New("no matching server")
	}

	if err := cache.Set(ctx, "runner-id:default", matchingServer.ID, -1); err != nil {
		return err
	}

	if len(matchingServer.Volumes) == 0 {
		return errors.New("no volume attached to the VM")
	}

	err = client.RenameVolume(ctx, conoha.RenameVolumeInput{
		VolumeID: matchingServer.Volumes[0].ID,
	})
	if err != nil {
		return err
	}

	return nil
}

func HandleEvent(ctx context.Context, runnerId string, strmService *streaming.StreamingService, cfg *config.Config, kvs *kvs.KeyValueStore, event *runner.Event) error {
	switch event.Type {
	case runner.EventHello:
		if event.Hello == nil {
			return errors.New("invalid event message: has no Hello")
		}
		if err := kvs.Set(ctx, fmt.Sprintf("runner-info:%s", runnerId), event.Hello, 30*24*time.Hour); err != nil {
			return err
		}

		if len(event.Hello.Addr.IPv4) != 0 {
			if err := AttachRunner(ctx, cfg, kvs, event.Hello.Addr.IPv4[0]); err != nil {
				slog.Error("Error updating runner ID", slog.Any("error", err))
			}
		}

	case runner.EventStatus:
		if event.Status == nil {
			return errors.New("invalid event message: has no Status")
		}

		strmService.PublishEvent(
			ctx,
			streaming.NewStandardMessageWithProgress(event.Status.EventCode, event.Status.Progress, GetPageCodeByEventCode(event.Status.EventCode)),
		)

	case runner.EventSysstat:
		if event.Sysstat == nil {
			return errors.New("invalid event message: has no Sysstat")
		}

		strmService.PublishEvent(
			ctx,
			streaming.NewSysstatMessage(event.Sysstat.CPUUsage, event.Sysstat.Time),
		)

	case runner.EventInfo:
		if event.Info == nil {
			return errors.New("invalid event message: has no Info")
		}

		strmService.PublishEvent(
			ctx,
			streaming.NewInfoMessage(event.Info.InfoCode, event.Info.IsError),
		)

	case runner.EventStarted:
		if event.Started == nil {
			return errors.New("invalid event message: has no Started")
		}

		if err := kvs.Set(ctx, fmt.Sprintf("world-info:%s", runnerId), event.Started, 30*24*time.Hour); err != nil {
			return err
		}
	}
	return nil
}

func GetSystemInfo(ctx context.Context, cfg *config.Config, cache *kvs.KeyValueStore) (*web.SystemInfo, error) {
	var serverHello runner.HelloExtra
	if err := cache.Get(ctx, "runner-info:default", &serverHello); err != nil {
		return nil, err
	}

	var ipAddr *string
	if len(serverHello.Addr.IPv4) != 0 {
		ipAddr = &serverHello.Addr.IPv4[0]
	}

	return &web.SystemInfo{
		PremisesVersion: serverHello.Version,
		HostOS:          serverHello.Host,
		IPAddress:       ipAddr,
	}, nil
}

func GetWorldInfo(ctx context.Context, cfg *config.Config, cache *kvs.KeyValueStore) (*web.WorldInfo, error) {
	var startedData runner.StartedExtra
	if err := cache.Get(ctx, "world-info:default", &startedData); err != nil {
		return nil, err
	}

	return &web.WorldInfo{
		Version:   startedData.ServerVersion,
		WorldName: startedData.World.Name,
		Seed:      startedData.World.Seed,
	}, nil
}
