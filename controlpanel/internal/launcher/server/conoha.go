package server

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
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
	conoha := conoha.NewClient(identity, endpoints, otelhttp.DefaultClient)
	return &ConohaServer{
		cfg:    cfg,
		conoha: conoha,
	}
}

func (s *ConohaServer) IsAvailable() bool {
	return s.cfg.ConohaUser != "" && s.cfg.ConohaPassword != ""
}

func findMatchingFlavor(flavors []conoha.Flavor, memSizeMB int) (string, error) {
	for _, f := range flavors {
		if f.RAM == memSizeMB {
			return f.ID, nil
		}
	}

	return "", errors.New("no matching flavor")
}

func findVolume(volumes []conoha.Volume, name string) (string, error) {
	for _, v := range volumes {
		if v.Name == name {
			return v.ID, nil
		}
	}

	return "", errors.New("no matching volume")
}

func isSupportedMemorySize(memSize int) bool {
	validMemSize := []int{1, 2, 4, 8, 16, 32, 64}
	return slices.Contains(validMemSize, memSize)
}

func getMemorySize(machineType string) (int, error) {
	memSizeGB, err := strconv.Atoi(strings.Replace(machineType, "g", "", 1))
	if err != nil {
		return 0, fmt.Errorf("invalid memory size: %w", err)
	} else if !isSupportedMemorySize(memSizeGB) {
		return 0, fmt.Errorf("unsupported memory size: %d", memSizeGB)
	}
	return memSizeGB, nil
}

func (s *ConohaServer) Start(ctx context.Context, gameConfig *runner.Config, machineType string) (ServerCookie, error) {
	memorySizeInGB, err := getMemorySize(machineType)
	if err != nil {
		return "", fmt.Errorf("invalid machine type: %w", err)
	}

	slog.Info("Retrieving flavors...")
	flavors, err := s.conoha.ListFlavorDetails(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to get flavors: %w", err)
	}
	flavor, err := findMatchingFlavor(flavors.Flavors, memorySizeInGB*1024)
	if err != nil {
		return "", fmt.Errorf("matching flavor not found: %w", err)
	}
	slog.Info("Retriving flavors...Done", slog.Any("selected_flavor", flavor))

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
		FlavorID:     flavor,
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
