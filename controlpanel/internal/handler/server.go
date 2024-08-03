package handler

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/kofuk/premises/controlpanel/internal/config"
	"github.com/kofuk/premises/controlpanel/internal/conoha"
	"github.com/kofuk/premises/internal/entity/runner"
	"github.com/kofuk/premises/internal/retry"
)

type GameServer struct {
	cfg     *config.Config
	conoha  *conoha.Client
	token   string
	expires string
	h       *Handler
}

func NewGameServer(cfg *config.Config, h *Handler) *GameServer {
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
	conoha := conoha.NewClient(identity, endpoints, nil)
	return &GameServer{
		cfg:    cfg,
		conoha: conoha,
		h:      h,
	}
}

func (s *GameServer) IsAvailable() bool {
	return s.cfg.ConohaUser != "" && s.cfg.ConohaPassword != ""
}

func findMatchingFlavor(flavors []conoha.Flavor, memSizeMB int) (string, error) {
	for _, f := range flavors {
		if f.RAM >= memSizeMB {
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

func (s *GameServer) SetUp(ctx context.Context, gameConfig *runner.Config, memSizeGB int, startupScript []byte) string {
	slog.Info("Retrieving flavors...")
	flavors, err := s.conoha.ListFlavorDetails(ctx)
	if err != nil {
		slog.Error("Failed to get flavors", slog.Any("error", err))
		return ""
	}
	flavor, err := findMatchingFlavor(flavors.Flavors, memSizeGB*1024)
	if err != nil {
		slog.Error("Matching flavor not found", slog.Any("error", err))
		return ""
	}
	slog.Info("Retriving flavors...Done", slog.Any("selected_flavor", flavor))

	slog.Info("Retriving volume ID...")
	volumes, err := s.conoha.ListVolumes(ctx)
	if err != nil {
		slog.Error("Failed to get volumes", slog.Any("error", err))
		return ""
	}
	volumeID, err := findVolume(volumes.Volumes, s.cfg.ConohaNameTag)
	if err != nil {
		slog.Error("Failed to get volume ID", slog.Any("error", err))
		return ""
	}
	slog.Info("Retriving image ID...Done", slog.String("volume_id", volumeID))

	slog.Info("Creating VM...")
	server, err := s.conoha.CreateServer(ctx, conoha.CreateServerInput{
		FlavorID:     flavor,
		RootVolumeID: volumeID,
		NameTag:      s.cfg.ConohaNameTag,
		UserData:     string(startupScript),
	})
	if err != nil {
		slog.Error("Failed to create server", slog.Any("error", err))
		return ""
	}
	slog.Info("Creating VM...")

	slog.Info("Waiting for VM to be created...")
	err = retry.Retry(func() error {
		server, err := s.conoha.GetServerDetail(ctx, conoha.GetServerDetailInput{
			ServerID: server.Server.ID,
		})
		if err != nil {
			slog.Info("Waiting for VM to be created...", slog.Any("error", err))
			return err
		} else if server.Server.Status == "BUILD" {
			slog.Info("Waiting for VM to be created...", slog.String("vm_status", server.Server.Status))
			return errors.New("VM is building")
		}

		return nil
	}, 30*time.Minute)
	if err != nil {
		slog.Error("Timeout creating VM", slog.Any("error", err))
		return ""
	}

	return server.Server.ID
}

func findServer(servers []conoha.ServerDetail, nameTag string) (string, error) {
	for _, s := range servers {
		if s.Metadata.InstanceNameTag == nameTag {
			return s.ID, nil
		}
	}

	return "", errors.New("no matching server")
}

func (s *GameServer) FindVM(ctx context.Context) (string, error) {
	servers, err := s.conoha.ListServerDetails(ctx)
	if err != nil {
		return "", err
	}

	serverID, err := findServer(servers.Servers, s.cfg.ConohaNameTag)
	if err != nil {
		return "", err
	}

	return serverID, nil
}

func (s *GameServer) VMRunning(ctx context.Context, id string) bool {
	slog.Info("Getting VM information...")
	server, err := s.conoha.GetServerDetail(ctx, conoha.GetServerDetailInput{
		ServerID: id,
	})
	if err != nil {
		return false
	}
	slog.Info("Getting VM information...Done")

	return server.Server.Status == "ACTIVE"
}

func (s *GameServer) StopVM(ctx context.Context, id string) bool {
	slog.Info("Requesting to Stop VM...")
	err := s.conoha.StopServer(ctx, conoha.StopServerInput{
		ServerID: id,
	})
	if err != nil {
		slog.Error("Failed to stop VM", slog.Any("error", err))
		return false
	}
	slog.Info("Requesting to Stop VM...Done")

	// Wait for VM to stop
	slog.Info("Waiting for the VM to stop...")
	err = retry.Retry(func() error {
		server, err := s.conoha.GetServerDetail(ctx, conoha.GetServerDetailInput{
			ServerID: id,
		})
		if err != nil {
			slog.Error("Failed to get VM information", slog.Any("error", err))
			return err
		}
		slog.Info("Waiting for the VM to stop...", slog.String("status", server.Server.Status))
		if server.Server.Status != "SHUTOFF" {
			return errors.New("not yet stopped")
		}

		return nil
	}, 30*time.Minute)
	if err != nil {
		slog.Error("Failed to stop VM", slog.Any("error", err))
		return false
	}
	slog.Info("Waiting for the VM to stop...Done")

	return true
}

func (s *GameServer) DeleteVM(ctx context.Context, id string) bool {
	slog.Info("Deleting VM...")
	err := s.conoha.DeleteServer(ctx, conoha.DeleteServerInput{
		ServerID: id,
	})
	if err != nil {
		slog.Error("Failed to delete VM", slog.Any("error", err))
		return false
	}
	slog.Info("Deleting VM...Done")

	return true
}
