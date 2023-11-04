package handler

import (
	"encoding/json"
	"net"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/kofuk/premises/controlpanel/config"
	"github.com/kofuk/premises/controlpanel/conoha"
	"github.com/kofuk/premises/controlpanel/gameconfig"
	"github.com/kofuk/premises/controlpanel/monitor"
	log "github.com/sirupsen/logrus"
)

type IPAddressSet struct {
	V4 net.IP
	V6 net.IP
}

type GameServer interface {
	SetUp(gameConfig *gameconfig.GameConfig, rdb *redis.Client, memSizeGB int) bool
	VMExists() bool
	VMRunning() bool
	StopVM(rdb *redis.Client) bool
	DeleteVM() bool
	ImageExists() bool
	SaveImage(rdb *redis.Client) bool
	DeleteImage(rdb *redis.Client) bool
	GetIPAddresses(rdb *redis.Client) *IPAddressSet
}

type ConohaServer struct {
	cfg     *config.Config
	token   string
	expires string
	h       *Handler
}

func NewConohaServer(cfg *config.Config, h *Handler) *ConohaServer {
	return &ConohaServer{
		cfg: cfg,
		h:   h,
	}
}

func (s *ConohaServer) getToken() (string, error) {
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

func (s *ConohaServer) SetUp(gameConfig *gameconfig.GameConfig, rdb *redis.Client, memSizeGB int) bool {
	locale := s.cfg.ControlPanel.Locale

	if err := monitor.PublishEvent(rdb, monitor.StatusData{
		Status: s.h.L(locale, "vm.gathering_info"),
	}); err != nil {
		log.WithError(err).Error("Failed to write status data to Redis channel")
	}

	token, err := s.getToken()
	if err != nil {
		log.WithError(err).Error("Failed to get token")
		return false
	}

	log.Info("Retrieving flavors...")
	flavors, err := conoha.GetFlavors(s.cfg, token)
	if err != nil {
		log.WithError(err).Error("Failed to get flavors")
		return false
	}
	flavorID := flavors.GetIDByMemSize(memSizeGB)
	log.WithField("selected_flavor", flavorID).Info("Retriving flavors...Done")

	log.Info("Retriving image ID...")
	imageID, imageStatus, err := conoha.GetImageID(s.cfg, token, s.cfg.Conoha.NameTag)
	if err != nil {
		log.WithError(err).Error("Failed to get image ID")
		return false
	} else if imageStatus != "active" {
		log.Error("Image is not active")
		return false
	}
	log.WithField("image_id", imageID).Info("Retriving image ID...Done")

	log.Info("Generating startup script...")
	gameConfigData, err := json.Marshal(gameConfig)
	if err != nil {
		log.WithError(err).Error("Failed to marshal config")
		return false
	}

	startupScript, err := conoha.GenerateStartupScript(gameConfigData, rdb)
	if err != nil {
		log.WithError(err).Error("Failed to generate startup script")
		return false
	}
	log.Info("Generating startup script...Done")

	log.Info("Creating VM...")
	if err := monitor.PublishEvent(rdb, monitor.StatusData{
		Status: s.h.L(locale, "vm.creating"),
	}); err != nil {
		log.WithError(err).Error("Failed to write status data to Redis channel")
	}
	if _, err := conoha.CreateVM(s.cfg, s.cfg.Conoha.NameTag, token, imageID, flavorID, startupScript); err != nil {
		log.WithError(err).Error("Failed to create VM")
		return false
	}
	log.Info("Creating VM...")

	log.Info("Waiting for VM to be created...")
	var vms *conoha.VMDetail
	for i := 0; i < 500; i++ {
		vms, err = conoha.GetVMDetail(s.cfg, token, s.cfg.Conoha.NameTag)
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
		if err := monitor.PublishEvent(rdb, monitor.StatusData{
			Status:   s.h.L(locale, "vm.get_detail.error"),
			HasError: true,
		}); err != nil {
			log.WithError(err).Error("Failed to write status data to Redis channel")
		}
		return false
	}
	log.WithField("vm_status", vms.Status).Info("Waiting for VM to be created...Done")

	ip4Addr := vms.GetIPAddress(4)
	ip6Addr := vms.GetIPAddress(6)
	log.WithField("ip_addr_4", ip4Addr).WithField("ip_addr_6", ip6Addr).Info("Got IP addresses")

	s.cfg.ServerAddr = ip4Addr

	return true
}

func (s *ConohaServer) VMExists() bool {
	token, err := s.getToken()
	if err != nil {
		log.WithError(err).Error("Failed to get token")
		return false
	}

	log.Info("Getting VM information...")
	detail, err := conoha.GetVMDetail(s.cfg, token, s.cfg.Conoha.NameTag)
	if err != nil {
		log.WithError(err).Error("Failed to get VM information")
		return false
	}
	log.Info("Getting VM information...Done")

	s.cfg.ServerAddr = detail.GetIPAddress(4)
	log.WithField("ip_addr", s.cfg.ServerAddr).Info("Stored IP address")

	return true
}

func (s *ConohaServer) VMRunning() bool {
	token, err := s.getToken()
	if err != nil {
		log.WithError(err).Error("Failed to get token")
		return false
	}

	log.Info("Getting VM information...")
	detail, err := conoha.GetVMDetail(s.cfg, token, s.cfg.Conoha.NameTag)
	if err != nil {
		log.WithError(err).Error("Failed to get VM detail")
		return false
	}
	log.Info("Getting VM information...Done")

	s.cfg.ServerAddr = detail.GetIPAddress(4)
	log.WithField("ip_addr", s.cfg.ServerAddr).Info("Stored IP address")

	return detail.Status == "ACTIVE"
}

func (s *ConohaServer) StopVM(rdb *redis.Client) bool {
	locale := s.cfg.ControlPanel.Locale

	token, err := s.getToken()
	if err != nil {
		log.WithError(err).Error("Failed to get token")
		return false
	}

	if err := monitor.PublishEvent(rdb, monitor.StatusData{
		Status: s.h.L(locale, "vm.stopping"),
	}); err != nil {
		log.WithError(err).Error("Failed to write status data to Redis channel")
	}

	log.Info("Getting VM information...")
	detail, err := conoha.GetVMDetail(s.cfg, token, s.cfg.Conoha.NameTag)
	if err != nil {
		log.WithError(err).Error("Failed to get VM detail")
		return false
	}
	log.Info("Getting VM information...Done")

	log.Info("Requesting to Stop VM...")
	if err := conoha.StopVM(s.cfg, token, detail.ID); err != nil {
		log.WithError(err).Error("Failed to stop VM")
		return false
	}
	log.Info("Requesting to Stop VM...Done")

	// Wait for VM to stop
	log.Info("Waiting for the VM to stop...")
	for {
		detail, err := conoha.GetVMDetail(s.cfg, token, s.cfg.Conoha.NameTag)
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

func (s *ConohaServer) DeleteVM() bool {
	token, err := s.getToken()
	if err != nil {
		log.WithError(err).Error("Failed to get token")
		return false
	}

	log.Info("Getting VM information...")
	detail, err := conoha.GetVMDetail(s.cfg, token, s.cfg.Conoha.NameTag)
	if err != nil {
		log.WithError(err).Error("Failed to get VM detail")
		return false
	}
	log.Info("Getting VM information...Done")

	log.Info("Deleting VM...")
	if err := conoha.DeleteVM(s.cfg, token, detail.ID); err != nil {
		log.WithError(err).Error("Failed to delete VM")
		return false
	}
	log.Info("Deleting VM...Done")

	return true
}

func (s *ConohaServer) ImageExists() bool {
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

func (s *ConohaServer) SaveImage(rdb *redis.Client) bool {
	locale := s.cfg.ControlPanel.Locale

	token, err := s.getToken()
	if err != nil {
		log.WithError(err).Error("Failed to get token")
		return false
	}

	if err := monitor.PublishEvent(rdb, monitor.StatusData{
		Status: s.h.L(locale, "vm.image.saving"),
	}); err != nil {
		log.WithError(err).Error("Failed to write status data to Redis channel")
	}

	log.Info("Getting VM information...")
	detail, err := conoha.GetVMDetail(s.cfg, token, s.cfg.Conoha.NameTag)
	if err != nil {
		log.WithError(err).Error("Failed to get VM detail")
		return false
	}
	log.Info("Getting VM information...Done")

	log.Info("Requesting to create image...")
	if err := conoha.CreateImage(s.cfg, token, detail.ID, s.cfg.Conoha.NameTag); err != nil {
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

func (s *ConohaServer) DeleteImage(rdb *redis.Client) bool {
	locale := s.cfg.ControlPanel.Locale

	token, err := s.getToken()
	if err != nil {
		log.WithError(err).Error("Failed to get token")
		return false
	}

	if err := monitor.PublishEvent(rdb, monitor.StatusData{
		Status: s.h.L(locale, "vm.cleaning_up"),
	}); err != nil {
		log.WithError(err).Error("Failed to write status data to Redis channel")
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

		if err := monitor.PublishEvent(rdb, monitor.StatusData{
			Status:   s.h.L(locale, "vm.image.delete.error"),
			HasError: true,
		}); err != nil {
			log.WithError(err).Error("Failed to write status data to Redis channel")
		}
		return false
	}
success:
	log.Info("Deleting image...Done")

	return true
}

func (s *ConohaServer) GetIPAddresses(rdb *redis.Client) *IPAddressSet {
	locale := s.cfg.ControlPanel.Locale

	token, err := s.getToken()
	if err != nil {
		log.WithError(err).Error("Failed to get token")
		return nil
	}

	if err := monitor.PublishEvent(rdb, monitor.StatusData{
		Status: s.h.L(locale, "vm.ip.waiting"),
	}); err != nil {
		log.WithError(err).Error("Failed to write status data to Redis channel")
	}

	log.Info("Getting VM information...")
	vms, err := conoha.GetVMDetail(s.cfg, token, s.cfg.Conoha.NameTag)
	if err != nil {
		log.WithField("vm_status", vms.Status).Info("Unable to get VM detail")
		return nil
	}
	log.Info("Getting VM information...Done")

	if err := monitor.PublishEvent(rdb, monitor.StatusData{
		Status: s.h.L(locale, "vm.dns.updating"),
	}); err != nil {
		log.WithError(err).Error("Failed to write status data to Redis channel")
	}

	result := IPAddressSet{
		V4: net.ParseIP(vms.GetIPAddress(4)),
		V6: net.ParseIP(vms.GetIPAddress(6)),
	}

	return &result
}
