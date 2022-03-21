package main

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"time"

	"github.com/kofuk/premises/cloudflare"
	"github.com/kofuk/premises/config"
	"github.com/kofuk/premises/conoha"
	"github.com/kofuk/premises/gameconfig"
	"github.com/kofuk/premises/monitor"
	log "github.com/sirupsen/logrus"
)

type GameServer interface {
	SetUp(gameConfig *gameconfig.GameConfig, memSizeGB int) bool
	VMExists() bool
	VMRunning() bool
	StopVM() bool
	DeleteVM() bool
	ImageExists() bool
	SaveImage() bool
	DeleteImage() bool
	UpdateDNS() bool
	RevertDNS() bool
}

type LocalDebugServer struct {
	pid int
	cfg *config.Config
}

func NewLocalDebugServer(cfg *config.Config) *LocalDebugServer {
	return &LocalDebugServer{
		cfg: cfg,
	}
}

func (s *LocalDebugServer) SetUp(gameConfig *gameconfig.GameConfig, memSizeGB int) bool {
	configData, err := json.Marshal(gameConfig)
	if err != nil {
		log.WithError(err).Error("Failed to marshal config")
		return false
	}
	if err := os.WriteFile(s.cfg.Locate("config.json"), configData, 0644); err != nil {
		log.WithError(err).Error("Failed to write config")
		return false
	}

	cmd := exec.Command("go", "run", ".")
	cmd.Dir = filepath.Join(os.Getenv("HOME"), "source/premises-mcmanager")
	cmd.Env = append(os.Environ(), "PREMISES_RUNNER_DEBUG=true")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		log.WithError(err).Error("Failed to start debug runner")
		return false
	}
	s.pid = cmd.Process.Pid
	s.cfg.ServerAddr = "localhost"

	go func() {
		if err := cmd.Wait(); err != nil {
			log.WithError(err).Error("Failed to wait command to finish")
		}
	}()

	return true
}

func (s *LocalDebugServer) SaveImage() bool {
	return true
}

func (s *LocalDebugServer) DeleteImage() bool {
	return true
}

func (s *LocalDebugServer) VMExists() bool {
	if s.pid != 0 {
		return true
	}
	return false
}

func (s *LocalDebugServer) VMRunning() bool {
	return s.VMExists()
}

func (s *LocalDebugServer) StopVM() bool {
	if err := syscall.Kill(-s.pid, syscall.SIGKILL); err != nil {
		log.WithError(err).Error("Failed to kill debug runner")
		return true
	}
	return true
}

func (s *LocalDebugServer) DeleteVM() bool {
	return true
}

func (s *LocalDebugServer) ImageExists() bool {
	return !s.VMExists()
}

func (s *LocalDebugServer) UpdateDNS() bool {
	return true
}

func (s *LocalDebugServer) RevertDNS() bool {
	return true
}

type ConohaServer struct {
	cfg     *config.Config
	token   string
	expires string
}

func NewConohaServer(cfg *config.Config) *ConohaServer {
	return &ConohaServer{
		cfg: cfg,
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

func (s *ConohaServer) SetUp(gameConfig *gameconfig.GameConfig, memSizeGB int) bool {
	locale := s.cfg.ControlPanel.Locale

	server.monitorChan <- &monitor.StatusData{
		Status: L(locale, "vm.gathering_info"),
	}

	token, err := s.getToken()
	if err != nil {
		log.WithError(err).Error("Failed to get token")
		return false
	}

	log.Info("Retriving flavors...")
	flavors, err := conoha.GetFlavors(s.cfg, token)
	if err != nil {
		log.WithError(err).Error("Failed to get flavors")
		return false
	}
	flavorID := flavors.GetIDByMemSize(memSizeGB)
	log.WithField("selected_flavor", flavorID).Info("Retriving flavors...Done")

	log.Info("Retriving image ID...")
	imageID, imageStatus, err := conoha.GetImageID(s.cfg, token, "mc-premises")
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

	startupScript, err := conoha.GenerateStartupScript(gameConfigData, s.cfg)
	if err != nil {
		log.WithError(err).Error("Failed to generate startup script")
		return false
	}
	log.Info("Generating startup script...Done")

	log.Info("Creating VM...")
	server.monitorChan <- &monitor.StatusData{
		Status: L(locale, "vm.creating"),
	}
	if _, err := conoha.CreateVM(s.cfg, token, imageID, flavorID, startupScript); err != nil {
		log.WithError(err).Error("Failed to create VM")
		return false
	}
	log.Info("Creating VM...")

	return true
}

func (s *ConohaServer) VMExists() bool {
	token, err := s.getToken()
	if err != nil {
		log.WithError(err).Error("Failed to get token")
		return false
	}

	log.Info("Getting VM information...")
	detail, err := conoha.GetVMDetail(s.cfg, token, "mc-premises")
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
	detail, err := conoha.GetVMDetail(s.cfg, token, "mc-premises")
	if err != nil {
		log.WithError(err).Error("Failed to get VM detail")
		return false
	}
	log.Info("Getting VM information...Done")

	s.cfg.ServerAddr = detail.GetIPAddress(4)
	log.WithField("ip_addr", s.cfg.ServerAddr).Info("Stored IP address")

	return detail.Status == "ACTIVE"
}

func (s *ConohaServer) StopVM() bool {
	locale := s.cfg.ControlPanel.Locale

	token, err := s.getToken()
	if err != nil {
		log.WithError(err).Error("Failed to get token")
		return false
	}

	server.monitorChan <- &monitor.StatusData{
		Status: L(locale, "vm.stopping"),
	}

	log.Info("Getting VM information...")
	detail, err := conoha.GetVMDetail(s.cfg, token, "mc-premises")
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
		time.Sleep(20 * time.Second)

		detail, err := conoha.GetVMDetail(s.cfg, token, "mc-premises")
		if err != nil {
			log.WithError(err).Error("Failed to get VM information")
			return false
		}
		log.WithField("status", detail.Status).Info("Waiting for the VM to stop...")
		if detail.Status == "SHUTOFF" {
			break
		}
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
	detail, err := conoha.GetVMDetail(s.cfg, token, "mc-premises")
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
	_, imageStatus, err := conoha.GetImageID(s.cfg, token, "mc-premises")
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

func (s *ConohaServer) SaveImage() bool {
	locale := s.cfg.ControlPanel.Locale

	token, err := s.getToken()
	if err != nil {
		log.WithError(err).Error("Failed to get token")
		return false
	}

	server.monitorChan <- &monitor.StatusData{
		Status: L(locale, "vm.image.saving"),
	}

	log.Info("Getting VM information...")
	detail, err := conoha.GetVMDetail(s.cfg, token, "mc-premises")
	if err != nil {
		log.WithError(err).Error("Failed to get VM detail")
		return false
	}
	log.Info("Getting VM information...Done")

	log.Info("Requesting to create image...")
	if err := conoha.CreateImage(s.cfg, token, detail.ID, "mc-premises"); err != nil {
		log.WithError(err).Error("Failed to create image")
		return false
	}
	log.Info("Requesting to create image...Done")

	log.Info("Waiting for image to be created...")
	for {
		_, imageStatus, err := conoha.GetImageID(s.cfg, token, "mc-premises")
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

func (s *ConohaServer) DeleteImage() bool {
	locale := s.cfg.ControlPanel.Locale

	token, err := s.getToken()
	if err != nil {
		log.WithError(err).Error("Failed to get token")
		return false
	}

	server.monitorChan <- &monitor.StatusData{
		Status: L(locale, "vm.cleaning_up"),
	}

	log.Info("Getting image information...")
	imageID, imageStatus, err := conoha.GetImageID(s.cfg, token, "mc-premises")
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

		server.monitorChan <- &monitor.StatusData{
			Status:   L(locale, "vm.image.delete.error"),
			HasError: true,
		}
		return false
	}
success:
	log.Info("Deleting image...Done")

	return true
}

func (s *ConohaServer) UpdateDNS() bool {
	locale := s.cfg.ControlPanel.Locale

	token, err := s.getToken()
	if err != nil {
		log.WithError(err).Error("Failed to get token")
		return false
	}

	server.monitorChan <- &monitor.StatusData{
		Status: L(locale, "vm.ip.waiting"),
	}

	log.Info("Getting VM information...")
	var vms *conoha.VMDetail
	for i := 0; i < 500; i++ {
		vms, err = conoha.GetVMDetail(s.cfg, token, "mc-premises")
		if err != nil || vms.Status == "BUILD" {
			log.WithField("vm_status", vms.Status).Info("Waiting VM to be created")
			time.Sleep(10 * time.Second)
			continue
		}
		break
	}

	if err != nil || vms.Status == "BUILD" {
		log.WithError(err).Error("Failed to get VM detail")
		if err == nil {
			log.Error("Building VM didn't completed")
		}
		server.monitorChan <- &monitor.StatusData{
			Status:   L(locale, "vm.get_detail.error"),
			HasError: true,
		}
		return false
	}
	log.WithField("vm_status", vms.Status).Info("Getting VM information...Done")

	server.monitorChan <- &monitor.StatusData{
		Status: L(locale, "vm.dns.updating"),
	}

	ip4Addr := vms.GetIPAddress(4)
	ip6Addr := vms.GetIPAddress(6)
	log.WithField("ip_addr_4", ip4Addr).WithField("ip_addr_6", ip6Addr).Info("Got IP addresses")

	s.cfg.ServerAddr = ip4Addr

	log.Info("Getting Cloudflare zone ID...")
	zoneID, err := cloudflare.GetZoneID(s.cfg)
	if err != nil {
		log.WithError(err).Error("Failed to get zone ID")
		server.monitorChan <- &monitor.StatusData{
			Status:   L(locale, "vm.dns.error"),
			HasError: true,
		}
		return false
	}
	log.WithField("zoen_id", zoneID).Info("Getting Cloudflare zone ID...Done")

	log.Info("Updating DNS record (v4)...")
	if err := cloudflare.UpdateDNS(s.cfg, zoneID, ip4Addr, 4); err != nil {
		log.WithError(err).Error("Failed to update DNS (v4)")
		server.monitorChan <- &monitor.StatusData{
			Status:   L(locale, "vm.dns.error"),
			HasError: true,
		}
		return false
	}
	log.Info("Updating DNS record (v4)...Done")

	log.Info("Updating DNS record (v6)...")
	if err := cloudflare.UpdateDNS(s.cfg, zoneID, ip6Addr, 6); err != nil {
		log.WithError(err).Error("Failed to update DNS (v6)")
		server.monitorChan <- &monitor.StatusData{
			Status:   L(locale, "vm.dns.error"),
			HasError: true,
		}
		return false
	}
	log.Info("Updating DNS record (v6)...Done")

	return true
}

func (s *ConohaServer) RevertDNS() bool {
	locale := s.cfg.ControlPanel.Locale

	server.monitorChan <- &monitor.StatusData{
		Status: L(locale, "vm.dns.reverting"),
	}

	log.Info("Getting Cloudflare zone ID...")
	zoneID, err := cloudflare.GetZoneID(s.cfg)
	if err != nil {
		log.WithError(err).Error("Failed to get zone ID")
		server.monitorChan <- &monitor.StatusData{
			Status:   L(locale, "vm.dns.error"),
			HasError: true,
		}
		return true
	}
	log.WithField("zoen_id", zoneID).Info("Getting Cloudflare zone ID...Done")

	log.Info("Updating DNS record (v4)...")
	if err := cloudflare.UpdateDNS(s.cfg, zoneID, "127.0.0.1", 4); err != nil {
		log.WithError(err).Error("Failed to update DNS (v4)")
		server.monitorChan <- &monitor.StatusData{
			Status:   L(locale, "vm.dns.error"),
			HasError: true,
		}
		return true
	}
	log.Info("Updating DNS record (v4)...Done")

	log.Info("Updating DNS record (v6)...")
	if err := cloudflare.UpdateDNS(s.cfg, zoneID, "::1", 6); err != nil {
		log.WithError(err).Error("Failed to update DNS (v6)")
		server.monitorChan <- &monitor.StatusData{
			Status:   L(locale, "vm.dns.error"),
			HasError: true,
		}
		return true
	}
	log.Info("Updating DNS record (v6)...Done")

	return true
}
