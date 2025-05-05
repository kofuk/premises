package conoha

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
	"github.com/kofuk/premises/controlpanel/internal/launcher/server"
	"github.com/kofuk/premises/controlpanel/internal/launcher/server/conoha/client"
	"github.com/kofuk/premises/controlpanel/internal/startup"
	"github.com/kofuk/premises/internal/entity/runner"
	"github.com/kofuk/premises/internal/retry"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

type ConohaServer struct {
	conoha   *client.Client
	user     string
	password string
	nameTag  string
}

var _ server.GameServer = (*ConohaServer)(nil)

func NewConohaServer(cfg *config.Config) server.GameServer {
	identity := client.Identity{
		User:     cfg.ConohaUser,
		Password: cfg.ConohaPassword,
		TenantID: cfg.ConohaTenantID,
	}
	endpoints := client.Endpoints{
		Identity: cfg.ConohaIdentityService,
		Compute:  cfg.ConohaComputeService,
		Image:    cfg.ConohaImageService,
		Volume:   cfg.ConohaVolumeService,
	}
	conoha := client.NewClient(identity, endpoints, otelhttp.DefaultClient)
	return &ConohaServer{
		conoha:   conoha,
		user:     cfg.ConohaUser,
		password: cfg.ConohaPassword,
		nameTag:  cfg.ConohaNameTag,
	}
}

func NewWithProviderSpecificData(ctx context.Context, data map[string]string) (server.GameServer, error) {
	user := data["user"]
	password := data["password"]

	identity := client.Identity{
		User:     user,
		Password: password,
		TenantID: data["tenant_id"],
	}
	endpoints := client.Endpoints{
		Identity: data["identity_service"],
		Compute:  data["compute_service"],
		Image:    data["image_service"],
		Volume:   data["volume_service"],
	}
	client := client.NewClient(identity, endpoints, otelhttp.DefaultClient)
	return &ConohaServer{
		conoha:   client,
		user:     user,
		password: password,
		nameTag:  data["name_tag"],
	}, nil
}

func (s *ConohaServer) IsAvailable(ctx context.Context) bool {
	return s.user != "" && s.password != ""
}

func findMatchingFlavor(flavors []client.Flavor, memSizeMB int) (string, error) {
	for _, f := range flavors {
		if f.RAM == memSizeMB {
			return f.ID, nil
		}
	}

	return "", errors.New("no matching flavor")
}

func findVolume(volumes []client.Volume, name string) (string, error) {
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

func (s *ConohaServer) Start(ctx context.Context, gameConfig *runner.Config, machineType string) (server.ServerCookie, error) {
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
	volumeID, err := findVolume(volumes.Volumes, s.nameTag)
	if err != nil {
		return "", fmt.Errorf("failed to get volume ID: %w", err)
	}
	slog.Info("Retriving image ID...Done", slog.String("volume_id", volumeID))

	slog.Info("Creating VM...")
	startupScript, _ := startup.GenerateStartupScript(gameConfig)
	sv, err := s.conoha.CreateServer(ctx, client.CreateServerInput{
		FlavorID:     flavor,
		RootVolumeID: volumeID,
		NameTag:      s.nameTag,
		UserData:     string(startupScript),
	})
	if err != nil {
		return "", fmt.Errorf("failed to create server: %w", err)
	}
	slog.Info("Creating VM...Done")

	slog.Info("Waiting for VM to be created...")
	_, err = retry.Retry(func() (retry.Void, error) {
		server, err := s.conoha.GetServerDetail(ctx, client.GetServerDetailInput{
			ServerID: sv.Server.ID,
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

	return server.ServerCookie(sv.Server.ID), nil
}

func findServer(servers []client.ServerDetail, nameTag string) (string, error) {
	for _, s := range servers {
		if s.Metadata.InstanceNameTag == nameTag {
			return s.ID, nil
		}
	}

	return "", errors.New("no matching server")
}

func (s *ConohaServer) Find(ctx context.Context) (server.ServerCookie, error) {
	servers, err := s.conoha.ListServerDetails(ctx)
	if err != nil {
		return "", err
	}

	serverID, err := findServer(servers.Servers, s.nameTag)
	if err != nil {
		return "", err
	}

	return server.ServerCookie(serverID), nil
}

func (s *ConohaServer) IsRunning(ctx context.Context, cookie server.ServerCookie) bool {
	slog.Info("Getting VM information...")
	server, err := s.conoha.GetServerDetail(ctx, client.GetServerDetailInput{
		ServerID: string(cookie),
	})
	if err != nil {
		return false
	}
	slog.Info("Getting VM information...Done")

	return server.Server.Status == "ACTIVE"
}

func (s *ConohaServer) Stop(ctx context.Context, cookie server.ServerCookie) bool {
	slog.Info("Requesting to Stop VM...")
	err := s.conoha.StopServer(ctx, client.StopServerInput{
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
		server, err := s.conoha.GetServerDetail(ctx, client.GetServerDetailInput{
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

func (s *ConohaServer) Delete(ctx context.Context, cookie server.ServerCookie) bool {
	slog.Info("Deleting VM...")
	err := s.conoha.DeleteServer(ctx, client.DeleteServerInput{
		ServerID: string(cookie),
	})
	if err != nil {
		slog.Error("Failed to delete VM", slog.Any("error", err))
		return false
	}
	slog.Info("Deleting VM...Done")

	return true
}
