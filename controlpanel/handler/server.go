package handler

import (
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/kofuk/premises/common/entity/runner"
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

func (s *GameServer) getToken() (string, error) {
	if s.token == "" {
		token, expires, err := conoha.GetToken(s.cfg)
		if err != nil {
			return "", err
		}
		s.token = token
		s.expires = expires
	} else {
		expires, err := time.Parse(time.RFC3339, s.expires)
		if err != nil || expires.Before(time.Now().Add(10*time.Minute)) {
			log.Info("Refreshing token...")
			token, expires, err := conoha.GetToken(s.cfg)
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

func (s *GameServer) SetUp(gameConfig *runner.Config, rdb *redis.Client, memSizeGB int, startupScript string) string {
	token, err := s.getToken()
	if err != nil {
		log.WithError(err).Error("Failed to get token")
		return ""
	}

	log.Info("Retrieving flavors...")
	flavors, err := conoha.GetFlavors(s.cfg, token)
	if err != nil {
		log.WithError(err).Error("Failed to get flavors")
		return ""
	}
	flavorID := flavors.GetIDByMemSize(memSizeGB)
	log.WithField("selected_flavor", flavorID).Info("Retriving flavors...Done")

	log.Info("Retriving image ID...")
	imageID, imageStatus, err := conoha.GetImageID(s.cfg, token, s.cfg.Conoha.NameTag)
	if err != nil {
		log.WithError(err).Error("Failed to get image ID")
		return ""
	} else if imageStatus != "active" {
		log.Error("Image is not active")
		return ""
	}
	log.WithField("image_id", imageID).Info("Retriving image ID...Done")

	log.Info("Creating VM...")
	id, err := conoha.CreateVM(s.cfg, s.cfg.Conoha.NameTag, token, imageID, flavorID, startupScript)
	if err != nil {
		log.WithError(err).Error("Failed to create VM")
		return ""
	}
	log.Info("Creating VM...")

	log.Info("Waiting for VM to be created...")
	var vms *conoha.VMDetail
	for i := 0; i < 500; i++ {
		vms, err = conoha.GetVMDetail(s.cfg, token, id)
		if err != nil {
			log.WithError(err).Info("Waiting for VM to be created...")
			time.Sleep(10 * time.Second)
			continue
		} else if vms.Status == "BUILD" {
			log.WithField("vm_status", vms.Status).Info("Waiting for VM to be created...")
			time.Sleep(10 * time.Second)
			continue
		}
		break
	}

	if err != nil || vms.Status == "BUILD" {
		log.WithError(err).Error("Timeout creating VM")
		if err == nil {
			log.Error("Building VM didn't completed")
		}
		return ""
	}
	log.WithField("vm_status", vms.Status).Info("Waiting for VM to be created...Done")

	return id
}

func (s *GameServer) FindVM() (string, error) {
	token, err := s.getToken()
	if err != nil {
		log.WithError(err).Error("Failed to get token")
		return "", err
	}

	detail, err := conoha.FindVM(s.cfg, token, conoha.FindByName(s.cfg.Conoha.NameTag))
	if err != nil {
		return "", err
	}

	return detail.ID, nil
}

func (s *GameServer) VMRunning(id string) bool {
	token, err := s.getToken()
	if err != nil {
		log.WithError(err).Error("Failed to get token")
		return false
	}

	log.Info("Getting VM information...")
	detail, err := conoha.GetVMDetail(s.cfg, token, id)
	if err != nil {
		log.WithError(err).Error("Failed to get VM detail")
		return false
	}
	log.Info("Getting VM information...Done")

	return detail.Status == "ACTIVE"
}

func (s *GameServer) StopVM(id string) bool {
	token, err := s.getToken()
	if err != nil {
		log.WithError(err).Error("Failed to get token")
		return false
	}

	log.Info("Requesting to Stop VM...")
	if err := conoha.StopVM(s.cfg, token, id); err != nil {
		log.WithError(err).Error("Failed to stop VM")
		return false
	}
	log.Info("Requesting to Stop VM...Done")

	// Wait for VM to stop
	log.Info("Waiting for the VM to stop...")
	for {
		detail, err := conoha.GetVMDetail(s.cfg, token, id)
		if err != nil {
			log.WithError(err).Error("Failed to get VM information")
			return false
		}
		log.WithField("status", detail.Status).Info("Waiting for the VM to stop...")
		if detail.Status == "SHUTOFF" {
			break
		}

		time.Sleep(20 * time.Second)
	}

	return true
}

func (s *GameServer) DeleteVM(id string) bool {
	token, err := s.getToken()
	if err != nil {
		log.WithError(err).Error("Failed to get token")
		return false
	}

	log.Info("Deleting VM...")
	if err := conoha.DeleteVM(s.cfg, token, id); err != nil {
		log.WithError(err).Error("Failed to delete VM")
		return false
	}
	log.Info("Deleting VM...Done")

	return true
}

func (s *GameServer) ImageExists() bool {
	token, err := s.getToken()
	if err != nil {
		log.WithError(err).Error("Failed to get token")
		return false
	}

	log.Info("Getting image information...")
	_, imageStatus, err := conoha.GetImageID(s.cfg, token, s.cfg.Conoha.NameTag)
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

func (s *GameServer) SaveImage(id string) bool {
	token, err := s.getToken()
	if err != nil {
		log.WithError(err).Error("Failed to get token")
		return false
	}

	log.Info("Requesting to create image...")
	if err := conoha.CreateImage(s.cfg, token, id, s.cfg.Conoha.NameTag); err != nil {
		log.WithError(err).Error("Failed to create image")
		return false
	}
	log.Info("Requesting to create image...Done")

	log.Info("Waiting for image to be created...")
	for {
		_, imageStatus, err := conoha.GetImageID(s.cfg, token, s.cfg.Conoha.NameTag)
		if err != nil {
			log.WithError(err).Error("Error getting image information; retrying...")
		} else if imageStatus == "active" {
			break
		}
		log.WithField("image_status", imageStatus).Info("Waiting for image to be created...")
		time.Sleep(30 * time.Second)
	}
	log.Info("Waiting for image to be created...Done")

	return true
}

func (s *GameServer) DeleteImage() bool {
	token, err := s.getToken()
	if err != nil {
		log.WithError(err).Error("Failed to get token")
		return false
	}

	log.Info("Getting image information...")
	imageID, imageStatus, err := conoha.GetImageID(s.cfg, token, s.cfg.Conoha.NameTag)
	if err != nil {
		log.WithError(err).Error("Failed to get image ID")
		return false
	} else if imageStatus != "active" {
		log.WithField("image_status", imageStatus).Error("Image is not active")
		return false
	}
	log.WithField("image_id", imageID).WithField("image_status", imageStatus).Info("Getting image information...Done")

	log.Info("Deleting image...")
	if err := conoha.DeleteImage(s.cfg, token, imageID); err != nil {
		log.WithError(err).Error("Seems we got undocumented response from image API; checking image existence...")
		for i := 0; i < 10; i++ {
			time.Sleep(2 * time.Second)
			if !s.ImageExists() {
				goto success
			}
			time.Sleep(3 * time.Second)
		}
		return false
	}
success:
	log.Info("Deleting image...Done")

	return true
}
