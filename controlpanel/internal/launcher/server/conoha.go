package server

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/kofuk/premises/controlpanel/internal/config"
	"github.com/kofuk/premises/controlpanel/internal/conoha"
	"github.com/kofuk/premises/controlpanel/internal/startup"
	"github.com/kofuk/premises/internal/entity/runner"
	"github.com/kofuk/premises/internal/retry"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

var availableFlavorsByMemory = map[int]string{
	2:   "g2l-t-c3m2",
	4:   "g2l-t-c4m4",
	12:  "g2l-t-c6m12",
	24:  "g2l-t-c8m24",
	48:  "g2l-t-c12m48",
	96:  "g2l-t-c24m96",
	128: "g2l-t-c40m128",
}

type ConohaServer struct {
	cfg    *config.Config
	conoha *conoha.Client
}

var _ GameServer = (*ConohaServer)(nil)

func NewConohaServer(cfg *config.Config) *ConohaServer {
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
	conoha := conoha.NewClient(identity, endpoints, &http.Client{
		Transport: otelhttp.NewTransport(http.DefaultTransport),
	})
	return &ConohaServer{
		cfg:    cfg,
		conoha: conoha,
	}
}

func (s *ConohaServer) IsAvailable() bool {
	return s.cfg.ConohaUser != "" && s.cfg.ConohaPassword != ""
}

func findVolume(volumes []conoha.Volume, name string) (string, error) {
	for _, v := range volumes {
		if v.Name == name {
			return v.ID, nil
		}
	}

	return "", errors.New("no matching volume")
}

func getMemorySize(machineType string) (string, error) {
	memSizeGB, err := strconv.Atoi(strings.Replace(machineType, "g", "", 1))
	if err != nil {
		return "", fmt.Errorf("invalid memory size: %w", err)
	} else if flavor, ok := availableFlavorsByMemory[memSizeGB]; !ok {
		return "", fmt.Errorf("unsupported memory size: %d", memSizeGB)
	} else {
		return flavor, nil
	}
}

func (s *ConohaServer) Start(ctx context.Context, gameConfig *runner.Config, machineType string) (ServerCookie, error) {
	flavorName, err := getMemorySize(machineType)
	if err != nil {
		return "", fmt.Errorf("invalid machine type: %w", err)
	}

	slog.Info("Retrieving flavors...")
	flavors, err := s.conoha.ListFlavorDetails(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to get flavors: %w", err)
	}
	flavorIndex := slices.IndexFunc(flavors.Flavors, func(e conoha.Flavor) bool { return e.Name == flavorName })
	if flavorIndex == -1 {
		return "", fmt.Errorf("matching flavor not found: %w", err)
	}
	flavorId := flavors.Flavors[flavorIndex].ID
	slog.Info("Retriving flavors...Done", slog.Any("selected_flavor", flavorId))

	slog.Info("Retriving volume ID...")
	volumes, err := s.conoha.ListVolumes(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to get volumes: %w", err)
	}
	volumeID, err := findVolume(volumes.Volumes, s.cfg.ConohaNameTag)
	if err != nil {
		return "", fmt.Errorf("failed to get volume ID: %w", err)
	}
	slog.Info("Retriving image ID...Done", slog.String("volume_id", volumeID))

	slog.Info("Creating VM...")
	startupScript, _ := startup.GenerateStartupScript(gameConfig)
	server, err := s.conoha.CreateServer(ctx, conoha.CreateServerInput{
		FlavorID:     flavorId,
		RootVolumeID: volumeID,
		NameTag:      s.cfg.ConohaNameTag,
		UserData:     string(startupScript),
	})
	if err != nil {
		return "", fmt.Errorf("failed to create server: %w", err)
	}
	slog.Info("Creating VM...Done")

	slog.Info("Waiting for VM to be created...")
	_, err = retry.Retry(func() (retry.Void, error) {
		server, err := s.conoha.GetServerDetail(ctx, conoha.GetServerDetailInput{
			ServerID: server.Server.ID,
		})
		if err != nil {
			slog.Info("Waiting for VM to be created...", slog.Any("error", err))
			return retry.V, err
		} else if server.Server.Status == "BUILD" {
			slog.Info("Waiting for VM to be created...", slog.String("vm_status", server.Server.Status))
			return retry.V, errors.New("VM is building")
		}

		return retry.V, nil
	}, 30*time.Minute)
	if err != nil {
		return "", fmt.Errorf("timeout creating VM: %w", err)
	}

	return ServerCookie(server.Server.ID), nil
}

func findServer(servers []conoha.ServerDetail, nameTag string) (string, error) {
	for _, s := range servers {
		if s.Metadata.InstanceNameTag == nameTag {
			return s.ID, nil
		}
	}

	return "", errors.New("no matching server")
}

func (s *ConohaServer) Find(ctx context.Context) (ServerCookie, error) {
	servers, err := s.conoha.ListServerDetails(ctx)
	if err != nil {
		return "", err
	}

	serverID, err := findServer(servers.Servers, s.cfg.ConohaNameTag)
	if err != nil {
		return "", err
	}

	return ServerCookie(serverID), nil
}

func (s *ConohaServer) IsRunning(ctx context.Context, cookie ServerCookie) bool {
	slog.Info("Getting VM information...")
	server, err := s.conoha.GetServerDetail(ctx, conoha.GetServerDetailInput{
		ServerID: string(cookie),
	})
	if err != nil {
		return false
	}
	slog.Info("Getting VM information...Done")

	return server.Server.Status == "ACTIVE"
}

func (s *ConohaServer) Stop(ctx context.Context, cookie ServerCookie) bool {
	slog.Info("Requesting to Stop VM...")
	err := s.conoha.StopServer(ctx, conoha.StopServerInput{
		ServerID: string(cookie),
	})
	if err != nil {
		slog.Error("Failed to stop VM", slog.Any("error", err))
		return false
	}
	slog.Info("Requesting to Stop VM...Done")

	// Wait for VM to stop
	slog.Info("Waiting for the VM to stop...")
	_, err = retry.Retry(func() (retry.Void, error) {
		server, err := s.conoha.GetServerDetail(ctx, conoha.GetServerDetailInput{
			ServerID: string(cookie),
		})
		if err != nil {
			slog.Error("Failed to get VM information", slog.Any("error", err))
			return retry.V, err
		}
		slog.Info("Waiting for the VM to stop...", slog.String("status", server.Server.Status))
		if server.Server.Status != "SHUTOFF" {
			return retry.V, errors.New("not yet stopped")
		}

		return retry.V, nil
	}, 30*time.Minute)
	if err != nil {
		slog.Error("Failed to stop VM", slog.Any("error", err))
		return false
	}
	slog.Info("Waiting for the VM to stop...Done")

	return true
}

func (s *ConohaServer) Delete(ctx context.Context, cookie ServerCookie) bool {
	slog.Info("Deleting VM...")
	err := s.conoha.DeleteServer(ctx, conoha.DeleteServerInput{
		ServerID: string(cookie),
	})
	if err != nil {
		slog.Error("Failed to delete VM", slog.Any("error", err))
		return false
	}
	slog.Info("Deleting VM...Done")

	return true
}
