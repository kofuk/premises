package handler

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/kofuk/premises/common/entity/runner"
	"github.com/kofuk/premises/common/retry"
	"github.com/kofuk/premises/controlpanel/config"
	"github.com/kofuk/premises/controlpanel/conoha"
	log "github.com/sirupsen/logrus"
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
			log.Info("Refreshing token...")
			token, expires, err := conoha.GetToken(ctx, s.cfg)
			if err != nil {
				return "", err
			}
			s.token = token
			s.expires = expires
			log.Info("Refreshing token...Done")
		}
	}

	return s.token, nil
}

func (s *GameServer) SetUp(ctx context.Context, gameConfig *runner.Config, memSizeGB int, startupScript []byte) string {
	token, err := s.getToken(ctx)
	if err != nil {
		log.WithError(err).Error("Failed to get token")
		return ""
	}

	log.Info("Finding security group...")
	sgs, err := conoha.GetSecurityGroups(ctx, s.cfg, token)
	if err != nil {
		log.WithError(err).Error("Failed to get security groups")
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
		log.Info("Createing security group...")
		sgId, err := conoha.CreateSecurityGroup(ctx, s.cfg, token, s.cfg.Conoha.NameTag)
		if err != nil {
			log.WithError(err).Error("Failed to create security group")
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
			log.WithError(err).Error("Failed to create security group rule")
			return ""
		}
		log.Info("Createing security group...Done")

	}
	log.Info("Finding security group...Done")

	log.Info("Retrieving flavors...")
	flavors, err := conoha.GetFlavors(ctx, s.cfg, token)
	if err != nil {
		log.WithError(err).Error("Failed to get flavors")
		return ""
	}
	flavorId, err := conoha.FindMatchingFlavor(flavors, memSizeGB*1024)
	if err != nil {
		log.WithError(err).Error("Matching flavor not found")
		return ""
	}

	log.WithField("selected_flavor", flavorId).Info("Retriving flavors...Done")

	log.Info("Retriving image ID...")
	imageID, imageStatus, err := conoha.GetImageID(ctx, s.cfg, token, s.cfg.Conoha.NameTag)
	if err != nil {
		log.WithError(err).Error("Failed to get image ID")
		return ""
	} else if imageStatus != "active" {
		log.Error("Image is not active")
		return ""
	}
	log.WithField("image_id", imageID).Info("Retriving image ID...Done")

	log.Info("Creating VM...")
	id, err := conoha.CreateVM(ctx, s.cfg, s.cfg.Conoha.NameTag, token, imageID, flavorId, startupScript)
	if err != nil {
		log.WithError(err).Error("Failed to create VM")
		return ""
	}
	log.Info("Creating VM...")

	log.Info("Waiting for VM to be created...")
	err = retry.Retry(func() error {
		vms, err := conoha.GetVMDetail(ctx, s.cfg, token, id)
		if err != nil {
			log.WithError(err).Info("Waiting for VM to be created...")
			return err
		} else if vms.Status == "BUILD" {
			log.WithField("vm_status", vms.Status).Info("Waiting for VM to be created...")
			return errors.New("VM is building")
		}

		return nil
	}, 30*time.Minute)
	if err != nil {
		log.WithError(err).Error("Timeout creating VM")
		return ""
	}

	return id
}

func (s *GameServer) FindVM(ctx context.Context) (string, error) {
	token, err := s.getToken(ctx)
	if err != nil {
		log.WithError(err).Error("Failed to get token")
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
		log.WithError(err).Error("Failed to get token")
		return false
	}

	log.Info("Getting VM information...")
	detail, err := conoha.GetVMDetail(ctx, s.cfg, token, id)
	if err != nil {
		log.WithError(err).Error("Failed to get VM detail")
		return false
	}
	log.Info("Getting VM information...Done")

	return detail.Status == "ACTIVE"
}

func (s *GameServer) StopVM(ctx context.Context, id string) bool {
	token, err := s.getToken(ctx)
	if err != nil {
		log.WithError(err).Error("Failed to get token")
		return false
	}

	log.Info("Requesting to Stop VM...")
	if err := conoha.StopVM(ctx, s.cfg, token, id); err != nil {
		log.WithError(err).Error("Failed to stop VM")
		return false
	}
	log.Info("Requesting to Stop VM...Done")

	// Wait for VM to stop
	log.Info("Waiting for the VM to stop...")
	err = retry.Retry(func() error {
		detail, err := conoha.GetVMDetail(ctx, s.cfg, token, id)
		if err != nil {
			log.WithError(err).Error("Failed to get VM information")
			return err
		}
		log.WithField("status", detail.Status).Info("Waiting for the VM to stop...")
		if detail.Status != "SHUTOFF" {
			return errors.New("Not yet stopped")
		}

		return nil
	}, 30*time.Minute)
	if err != nil {
		log.WithError(err).Error("Failed to stop VM")
		return false
	}
	log.Info("Waiting for the VM to stop...Done")

	return true
}

func (s *GameServer) DeleteVM(ctx context.Context, id string) bool {
	token, err := s.getToken(ctx)
	if err != nil {
		log.WithError(err).Error("Failed to get token")
		return false
	}

	log.Info("Deleting VM...")
	if err := conoha.DeleteVM(ctx, s.cfg, token, id); err != nil {
		log.WithError(err).Error("Failed to delete VM")
		return false
	}
	log.Info("Deleting VM...Done")

	return true
}

func (s *GameServer) ImageExists(ctx context.Context) bool {
	token, err := s.getToken(ctx)
	if err != nil {
		log.WithError(err).Error("Failed to get token")
		return false
	}

	log.Info("Getting image information...")
	_, imageStatus, err := conoha.GetImageID(ctx, s.cfg, token, s.cfg.Conoha.NameTag)
	if err != nil {
		log.WithError(err).Error("Failed to get image information")
		return false
	} else if imageStatus != "active" {
		log.Info("Getting image information...Done")
		log.WithField("status", imageStatus).Info("Image is not active")
		return false
	}
	log.Info("Getting image information...Done")

	return true
}

func (s *GameServer) SaveImage(ctx context.Context, id string) bool {
	token, err := s.getToken(ctx)
	if err != nil {
		log.WithError(err).Error("Failed to get token")
		return false
	}

	log.Info("Retrieving to volume information...")
	vm, err := conoha.GetVMDetail(ctx, s.cfg, token, id)
	if err != nil {
		log.WithError(err).Error("Error retrieving VM detail")
		return false
	}
	if len(vm.Volumes) == 0 {
		log.Error("No volume attached to the VM")
		return false
	}
	log.Info("Retrieving to volume information...Done")

	log.Info("Requesting to create image...")
	if err := conoha.CreateImage(ctx, s.cfg, token, vm.Volumes[0].ID, s.cfg.Conoha.NameTag); err != nil {
		log.WithError(err).Error("Failed to create image")
		return false
	}
	log.Info("Requesting to create image...Done")

	log.Info("Waiting for image to be created...")
	err = retry.Retry(func() error {
		_, imageStatus, err := conoha.GetImageID(ctx, s.cfg, token, s.cfg.Conoha.NameTag)
		if err != nil {
			log.WithError(err).Error("Error getting image information")
		}
		if imageStatus == "active" {
			return nil
		}
		log.WithField("image_status", imageStatus).Info("Waiting for image to be created...")
		return fmt.Errorf("Image is not active (status=%s)", imageStatus)
	}, 30*time.Minute)
	if err != nil {
		log.WithError(err).Error("Failed save image")
		return false
	}
	log.Info("Waiting for image to be created...Done")

	return true
}

func (s *GameServer) DeleteImage(ctx context.Context) bool {
	token, err := s.getToken(ctx)
	if err != nil {
		log.WithError(err).Error("Failed to get token")
		return false
	}

	log.Info("Getting image information...")
	imageID, imageStatus, err := conoha.GetImageID(ctx, s.cfg, token, s.cfg.Conoha.NameTag)
	if err != nil {
		log.WithError(err).Error("Failed to get image ID")
		return false
	} else if imageStatus != "active" {
		log.WithField("image_status", imageStatus).Error("Image is not active")
		return false
	}
	log.WithField("image_id", imageID).WithField("image_status", imageStatus).Info("Getting image information...Done")

	log.Info("Deleting image...")
	if err := conoha.DeleteImage(ctx, s.cfg, token, imageID); err != nil {
		log.WithError(err).Error("Seems we got undocumented response from image API; checking image existence...")
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
	log.Info("Deleting image...Done")

	return true
}
