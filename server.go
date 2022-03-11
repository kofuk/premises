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
	if err := os.WriteFile(filepath.Join(s.cfg.Prefix, "/opt/premises/config.json"), configData, 0644); err != nil {
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
			token, expires, err := conoha.GetToken(s.cfg)
			if err != nil {
				return "", err
			}
			s.token = token
			s.expires = expires
		}
	}

	return s.token, nil
}

func (s *ConohaServer) SetUp(gameConfig *gameconfig.GameConfig, memSizeGB int) bool {
	server.monitorChan <- &monitor.StatusData{
		Status: "Gathering information...",
	}

	token, err := s.getToken()
	if err != nil {
		log.WithError(err).Error("Failed to get token")
		return false
	}

	flavors, err := conoha.GetFlavors(s.cfg, token)
	if err != nil {
		log.WithError(err).Error("Failed to get flavors")
		return false
	}
	flavorID := flavors.GetIDByMemSize(memSizeGB)

	imageID, imageStatus, err := conoha.GetImageID(s.cfg, token, "mc-premises")
	if err != nil {
		log.WithError(err).Error("Failed to get image ID")
		return false
	} else if imageStatus != "active" {
		log.Error("Image is not active")
		return false
	}

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

	server.monitorChan <- &monitor.StatusData{
		Status: "Creating VM...",
	}

	if _, err := conoha.CreateVM(s.cfg, token, imageID, flavorID, startupScript); err != nil {
		log.WithError(err).Error("Failed to create VM")
		return false
	}

	return true
}

func (s *ConohaServer) VMExists() bool {
	token, err := s.getToken()
	if err != nil {
		log.WithError(err).Error("Failed to get token")
		return false
	}

	detail, err := conoha.GetVMDetail(s.cfg, token, "mc-premises")
	if err != nil {
		return false
	}

	s.cfg.ServerAddr = detail.GetIPAddress(4)

	return true
}

func (s *ConohaServer) VMRunning() bool {
	token, err := s.getToken()
	if err != nil {
		log.WithError(err).Error("Failed to get token")
		return false
	}
	detail, err := conoha.GetVMDetail(s.cfg, token, "mc-premises")
	if err != nil {
		log.WithError(err).Error("Failed to get VM detail")
		return false
	}

	s.cfg.ServerAddr = detail.GetIPAddress(4)

	return detail.Status == "ACTIVE"
}

func (s *ConohaServer) StopVM() bool {
	token, err := s.getToken()
	if err != nil {
		log.WithError(err).Error("Failed to get token")
		return false
	}

	detail, err := conoha.GetVMDetail(s.cfg, token, "mc-premises")
	if err != nil {
		log.WithError(err).Error("Failed to get VM detail")
		return false
	}

	if err := conoha.StopVM(s.cfg, token, detail.ID); err != nil {
		log.WithError(err).Error("Failed to stop VM")
		return false
	}

	// Wait for VM to stop
	for {
		time.Sleep(20 * time.Second)

		detail, err := conoha.GetVMDetail(s.cfg, token, "mc-premises")
		if err != nil {
			return false
		}
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

	detail, err := conoha.GetVMDetail(s.cfg, token, "mc-premises")
	if err != nil {
		log.WithError(err).Error("Failed to get VM detail")
		return false
	}

	if err := conoha.DeleteVM(s.cfg, token, detail.ID); err != nil {
		log.WithError(err).Error("Failed to delete VM")
		return false
	}

	return true
}

func (s *ConohaServer) ImageExists() bool {
	token, err := s.getToken()
	if err != nil {
		log.WithError(err).Error("Failed to get token")
		return false
	}

	_, imageStatus, err := conoha.GetImageID(s.cfg, token, "mc-premises")
	if err != nil {
		log.WithError(err).Error("Failed to get token")
		return false
	} else if imageStatus != "active" {
		log.Info("Image is not active")
		return false
	}

	return true
}

func (s *ConohaServer) SaveImage() bool {
	token, err := s.getToken()
	if err != nil {
		log.WithError(err).Error("Failed to get token")
		return false
	}

	detail, err := conoha.GetVMDetail(s.cfg, token, "mc-premises")
	if err != nil {
		log.WithError(err).Error("Failed to get VM detail")
		return false
	}

	if err := conoha.CreateImage(s.cfg, token, detail.ID, "mc-premises"); err != nil {
		log.WithError(err).Error("Failed to create image")
		return false
	}

	for {
		if _, imageStatus, err := conoha.GetImageID(s.cfg, token, "mc-premises"); err == nil && imageStatus == "active" {
			break
		}
		time.Sleep(30 * time.Second)
	}

	return true
}

func (s *ConohaServer) DeleteImage() bool {
	token, err := s.getToken()
	if err != nil {
		log.WithError(err).Error("Failed to get token")
		return false
	}

	server.monitorChan <- &monitor.StatusData{
		Status: "Cleaning up...",
	}

	imageID, imageStatus, err := conoha.GetImageID(s.cfg, token, "mc-premises")
	if err != nil {
		log.WithError(err).Error("Failed to get image ID")
		return false
	} else if imageStatus != "active" {
		log.Error("Image is not active")
		return false
	}

	if err := conoha.DeleteImage(s.cfg, token, imageID); err != nil {
		log.WithError(err).Error("Failed to delete image")

		server.monitorChan <- &monitor.StatusData{
			Status:   "Failed to outdated image",
			HasError: true,
		}
		return false
	}

	return true
}

func (s *ConohaServer) UpdateDNS() bool {
	token, err := s.getToken()
	if err != nil {
		log.WithError(err).Error("Failed to get token")
		return false
	}

	server.monitorChan <- &monitor.StatusData{
		Status: "Obtaining IP address...",
	}

	var vms *conoha.VMDetail
	for i := 0; i < 10; i++ {
		vms, err = conoha.GetVMDetail(s.cfg, token, "mc-premises")
		if err != nil || vms.Status == "BUILD" {
			log.Info("Waiting VM to be created")
			time.Sleep(10 * time.Second)
			continue
		}
		break
	}

	if err != nil {
		log.WithError(err).Error("Failed to get VM detail")
		server.monitorChan <- &monitor.StatusData{
			Status:   "Failed to get information on created VM",
			HasError: true,
		}
		return false
	}

	server.monitorChan <- &monitor.StatusData{
		Status: "Updating DNS records...",
	}

	ip4Addr := vms.GetIPAddress(4)
	ip6Addr := vms.GetIPAddress(6)

	s.cfg.ServerAddr = ip4Addr

	zoneID, err := cloudflare.GetZoneID(s.cfg)
	if err != nil {
		log.WithError(err).Error("Failed to get zone ID")
		server.monitorChan <- &monitor.StatusData{
			Status:   "Failed to update DNS records",
			HasError: true,
		}
		return false
	}

	if err := cloudflare.UpdateDNS(s.cfg, zoneID, ip4Addr, 4); err != nil {
		log.WithError(err).Error("Failed to update DNS (v4)")
		server.monitorChan <- &monitor.StatusData{
			Status:   "Failed to update DNS records",
			HasError: true,
		}
		return false
	}

	if err := cloudflare.UpdateDNS(s.cfg, zoneID, ip6Addr, 6); err != nil {
		log.WithError(err).Error("Failed to update DNS (v6)")
		server.monitorChan <- &monitor.StatusData{
			Status:   "Failed to update DNS records",
			HasError: true,
		}
		return false
	}

	return true
}

func (s *ConohaServer) RevertDNS() bool {
	server.monitorChan <- &monitor.StatusData{
		Status: "Reverting DNS records...",
	}

	zoneID, err := cloudflare.GetZoneID(s.cfg)
	if err != nil {
		log.WithError(err).Error("Failed to get zone ID")
		server.monitorChan <- &monitor.StatusData{
			Status:   "Failed to update DNS records",
			HasError: true,
		}
		return true
	}

	if err := cloudflare.UpdateDNS(s.cfg, zoneID, "127.0.0.1", 4); err != nil {
		log.WithError(err).Error("Failed to update DNS (v4)")
		server.monitorChan <- &monitor.StatusData{
			Status:   "Failed to update DNS records",
			HasError: true,
		}
		return true
	}

	if err := cloudflare.UpdateDNS(s.cfg, zoneID, "::1", 6); err != nil {
		log.WithError(err).Error("Failed to update DNS (v6)")
		server.monitorChan <- &monitor.StatusData{
			Status:   "Failed to update DNS records",
			HasError: true,
		}
		return true
	}

	return true
}
