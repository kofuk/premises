package monitor

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
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

	if len(vm.Volumes) == 0 {
		return errors.New("no volume attached to the VM")
	}

	if err := conoha.RenameVolume(ctx, cfg, token, vm.Volumes[0].ID, cfg.ConohaNameTag); err != nil {
		return err
	}

	return nil
}

func HandleEvent(ctx context.Context, runnerId string, strmProvider *streaming.StreamingService, cfg *config.Config, kvs *kvs.KeyValueStore, event *runner.Event) error {
	stdStream := strmProvider.GetStream(streaming.StandardStream)
	infoStream := strmProvider.GetStream(streaming.InfoStream)
	sysstatStream := strmProvider.GetStream(streaming.SysstatStream)

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

			url, _ := url.Parse(cfg.ProxyAPIEndpoint)
			url.Path = "/set"
			q := url.Query()
			q.Add("name", cfg.GameDomain)
			q.Add("addr", event.Hello.Addr.IPv4[0]+":25565")
			url.RawQuery = q.Encode()

			resp, err := http.Post(url.String(), "text/plain", nil)
			if err != nil {
				slog.Error("Error updating proxy", slog.Any("error", err))
			} else {
				io.Copy(io.Discard, resp.Body)
			}
		}

	case runner.EventStatus:
		if event.Status == nil {
			return errors.New("invalid event message: has no Status")
		}

		if err := strmProvider.PublishEvent(
			ctx,
			stdStream,
			streaming.NewStandardMessageWithProgress(event.Status.EventCode, event.Status.Progress, GetPageCodeByEventCode(event.Status.EventCode)),
		); err != nil {
			return err
		}

	case runner.EventSysstat:
		if event.Sysstat == nil {
			return errors.New("invalid event message: has no Sysstat")
		}

		if err := strmProvider.PublishEvent(
			ctx,
			sysstatStream,
			streaming.NewSysstatMessage(event.Sysstat.CPUUsage, event.Sysstat.Time),
		); err != nil {
			return err
		}

	case runner.EventInfo:
		if event.Info == nil {
			return errors.New("invalid event message: has no Info")
		}

		if err := strmProvider.PublishEvent(
			ctx,
			infoStream,
			streaming.NewInfoMessage(event.Info.InfoCode, event.Info.IsError),
		); err != nil {
			return err
		}

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
