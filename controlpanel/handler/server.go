package handler

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/kofuk/premises/common/entity/runner"
	"github.com/kofuk/premises/common/retry"
	"github.com/kofuk/premises/controlpanel/config"
	"github.com/kofuk/premises/controlpanel/conoha"
)

type GameServer struct {
	cfg     *config.Config
	token   string
	expires string
	h       *Handler
}

func NewGameServer(cfg *config.Config, h *Handler) *GameServer {
	return &GameServer{
		cfg: cfg,
		h:   h,
	}
}

func (s *GameServer) getToken(ctx context.Context) (string, error) {
	if s.token == "" {
		token, expires, err := conoha.GetToken(ctx, s.cfg)
		if err != nil {
			return "", err
		}
		s.token = token
		s.expires = expires
	} else {
		expires, err := time.Parse(time.RFC3339, s.expires)
		if err != nil || expires.Before(time.Now().Add(10*time.Minute)) {
			slog.Info("Refreshing token...")
			token, expires, err := conoha.GetToken(ctx, s.cfg)
			if err != nil {
				return "", err
			}
			s.token = token
			s.expires = expires
			slog.Info("Refreshing token...Done")
		}
	}

	return s.token, nil
}

func (s *GameServer) SetUp(ctx context.Context, gameConfig *runner.Config, memSizeGB int, startupScript []byte) string {
	token, err := s.getToken(ctx)
	if err != nil {
		slog.Error("Failed to get token", slog.Any("error", err))
		return ""
	}

	slog.Info("Finding security group...")
	sgs, err := conoha.GetSecurityGroups(ctx, s.cfg, token)
	if err != nil {
		slog.Error("Failed to get security groups", slog.Any("error", err))
		return ""
	}
	hasSecGroup := false
	for _, sg := range sgs {
		if sg.Name == s.cfg.Conoha.NameTag {
			hasSecGroup = true
			break
		}
	}
	if !hasSecGroup {
		slog.Info("Createing security group...")
		sgId, err := conoha.CreateSecurityGroup(ctx, s.cfg, token, s.cfg.Conoha.NameTag)
		if err != nil {
			slog.Error("Failed to create security group", slog.Any("error", err))
			return ""
		}
		if err := conoha.CreateSecurityGroupRule(ctx, s.cfg, token, conoha.SecurityGroupRule{
			SecurityGroupID: sgId,
			Direction:       "ingress",
			EtherType:       "IPv4",
			PortRangeMin:    "25565",
			PortRangeMax:    "25565",
			Protocol:        "tcp",
			RemoteIP:        "0.0.0.0/0",
		}); err != nil {
			slog.Error("Failed to create security group rule", slog.Any("error", err))
			return ""
		}
		slog.Info("Createing security group...Done")

	}
	slog.Info("Finding security group...Done")

	slog.Info("Retrieving flavors...")
	flavors, err := conoha.GetFlavors(ctx, s.cfg, token)
	if err != nil {
		slog.Error("Failed to get flavors", slog.Any("error", err))
		return ""
	}
	flavorId, err := conoha.FindMatchingFlavor(flavors, memSizeGB*1024)
	if err != nil {
		slog.Error("Matching flavor not found", slog.Any("error", err))
		return ""
	}

	slog.Info("Retriving flavors...Done", slog.String("selected_flavor", flavorId))

	slog.Info("Retriving image ID...")
	imageID, imageStatus, err := conoha.GetImageID(ctx, s.cfg, token, s.cfg.Conoha.NameTag)
	if err != nil {
		slog.Error("Failed to get image ID", slog.Any("error", err))
		return ""
	} else if imageStatus != "active" {
		slog.Error("Image is not active")
		return ""
	}
	slog.Info("Retriving image ID...Done", slog.String("image_id", imageID))

	slog.Info("Creating VM...")
	id, err := conoha.CreateVM(ctx, s.cfg, s.cfg.Conoha.NameTag, token, imageID, flavorId, startupScript)
	if err != nil {
		slog.Error("Failed to create VM", slog.Any("error", err))
		return ""
	}
	slog.Info("Creating VM...")

	slog.Info("Waiting for VM to be created...")
	err = retry.Retry(func() error {
		vms, err := conoha.GetVMDetail(ctx, s.cfg, token, id)
		if err != nil {
			slog.Info("Waiting for VM to be created...", slog.Any("error", err))
			return err
		} else if vms.Status == "BUILD" {
			slog.Info("Waiting for VM to be created...", slog.String("vm_status", vms.Status))
			return errors.New("VM is building")
		}

		return nil
	}, 30*time.Minute)
	if err != nil {
		slog.Error("Timeout creating VM", slog.Any("error", err))
		return ""
	}

	return id
}

func (s *GameServer) FindVM(ctx context.Context) (string, error) {
	token, err := s.getToken(ctx)
	if err != nil {
		slog.Error("Failed to get token", slog.Any("error", err))
		return "", err
	}

	detail, err := conoha.FindVM(ctx, s.cfg, token, conoha.FindByName(s.cfg.Conoha.NameTag))
	if err != nil {
		return "", err
	}

	return detail.ID, nil
}

func (s *GameServer) VMRunning(ctx context.Context, id string) bool {
	token, err := s.getToken(ctx)
	if err != nil {
		slog.Error("Failed to get token", slog.Any("error", err))
		return false
	}

	slog.Info("Getting VM information...")
	detail, err := conoha.GetVMDetail(ctx, s.cfg, token, id)
	if err != nil {
		slog.Error("Failed to get VM detail", slog.Any("error", err))
		return false
	}
	slog.Info("Getting VM information...Done")

	return detail.Status == "ACTIVE"
}

func (s *GameServer) StopVM(ctx context.Context, id string) bool {
	token, err := s.getToken(ctx)
	if err != nil {
		slog.Error("Failed to get token", slog.Any("error", err))
		return false
	}

	slog.Info("Requesting to Stop VM...")
	if err := conoha.StopVM(ctx, s.cfg, token, id); err != nil {
		slog.Error("Failed to stop VM", slog.Any("error", err))
		return false
	}
	slog.Info("Requesting to Stop VM...Done")

	// Wait for VM to stop
	slog.Info("Waiting for the VM to stop...")
	err = retry.Retry(func() error {
		detail, err := conoha.GetVMDetail(ctx, s.cfg, token, id)
		if err != nil {
			slog.Error("Failed to get VM information", slog.Any("error", err))
			return err
		}
		slog.Info("Waiting for the VM to stop...", slog.String("status", detail.Status))
		if detail.Status != "SHUTOFF" {
			return errors.New("Not yet stopped")
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
	token, err := s.getToken(ctx)
	if err != nil {
		slog.Error("Failed to get token", slog.Any("error", err))
		return false
	}

	slog.Info("Deleting VM...")
	if err := conoha.DeleteVM(ctx, s.cfg, token, id); err != nil {
		slog.Error("Failed to delete VM", slog.Any("error", err))
		return false
	}
	slog.Info("Deleting VM...Done")

	return true
}

func (s *GameServer) ImageExists(ctx context.Context) bool {
	token, err := s.getToken(ctx)
	if err != nil {
		slog.Error("Failed to get token", slog.Any("error", err))
		return false
	}

	slog.Info("Getting image information...")
	_, imageStatus, err := conoha.GetImageID(ctx, s.cfg, token, s.cfg.Conoha.NameTag)
	if err != nil {
		slog.Error("Failed to get image information", slog.Any("error", err))
		return false
	} else if imageStatus != "active" {
		slog.Info("Getting image information...Done")
		slog.Info("Image is not active", slog.String("status", imageStatus))
		return false
	}
	slog.Info("Getting image information...Done")

	return true
}

func (s *GameServer) SaveImage(ctx context.Context, id string) bool {
	token, err := s.getToken(ctx)
	if err != nil {
		slog.Error("Failed to get token", slog.Any("error", err))
		return false
	}

	slog.Info("Retrieving to volume information...")
	vm, err := conoha.GetVMDetail(ctx, s.cfg, token, id)
	if err != nil {
		slog.Error("Error retrieving VM detail", slog.Any("error", err))
		return false
	}
	if len(vm.Volumes) == 0 {
		slog.Error("No volume attached to the VM")
		return false
	}
	slog.Info("Retrieving to volume information...Done")

	slog.Info("Requesting to create image...")
	if err := conoha.CreateImage(ctx, s.cfg, token, vm.Volumes[0].ID, s.cfg.Conoha.NameTag); err != nil {
		slog.Error("Failed to create image", slog.Any("error", err))
		return false
	}
	slog.Info("Requesting to create image...Done")

	slog.Info("Waiting for image to be created...")
	err = retry.Retry(func() error {
		_, imageStatus, err := conoha.GetImageID(ctx, s.cfg, token, s.cfg.Conoha.NameTag)
		if err != nil {
			slog.Error("Error getting image information", slog.Any("error", err))
		}
		if imageStatus == "active" {
			return nil
		}
		slog.Info("Waiting for image to be created...", slog.String("image_status", imageStatus))
		return fmt.Errorf("Image is not active (status=%s)", imageStatus)
	}, 30*time.Minute)
	if err != nil {
		slog.Error("Failed save image", slog.Any("error", err))
		return false
	}
	slog.Info("Waiting for image to be created...Done")

	return true
}

func (s *GameServer) DeleteImage(ctx context.Context) bool {
	token, err := s.getToken(ctx)
	if err != nil {
		slog.Error("Failed to get token", slog.Any("error", err))
		return false
	}

	slog.Info("Getting image information...")
	imageID, imageStatus, err := conoha.GetImageID(ctx, s.cfg, token, s.cfg.Conoha.NameTag)
	if err != nil {
		slog.Error("Failed to get image ID", slog.Any("error", err))
		return false
	} else if imageStatus != "active" {
		slog.Error("Image is not active", slog.String("image_status", imageStatus))
		return false
	}
	slog.Info("Getting image information...Done", slog.String("image_id", imageID), slog.String("image_status", imageStatus))

	slog.Info("Deleting image...")
	if err := conoha.DeleteImage(ctx, s.cfg, token, imageID); err != nil {
		slog.Error("Seems we got undocumented response from image API; checking image existence...", slog.Any("error", err))
		err := retry.Retry(func() error {
			if !s.ImageExists(ctx) {
				return nil
			}

			return errors.New("Image exists")
		}, 1*time.Minute)
		if err != nil {
			return false
		}
	}
	slog.Info("Deleting image...Done")

	return true
}
