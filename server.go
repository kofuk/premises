package main

import (
	"encoding/json"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"

	"github.com/kofuk/premises/config"
	"github.com/kofuk/premises/gameconfig"
)

type GameServer interface {
	SetUp(gameConfig *gameconfig.GameConfig) bool
	VMExists() bool
	VMRunning() bool
	StopVM() bool
	DeleteVM() bool
	ImageExists() bool
	SaveImage() bool
	DeleteImage() bool
	UpdateDNS() bool
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

func (s *LocalDebugServer) SetUp(gameConfig *gameconfig.GameConfig) bool {
	configData, err := json.Marshal(gameConfig)
	if err != nil {
		log.Println(err)
		return false
	}
	if err := os.WriteFile(filepath.Join(s.cfg.Prefix, "/opt/premises/config.json"), configData, 0644); err != nil {
		log.Println(err)
		return false
	}

	cmd := exec.Command("go", "run", ".")
	cmd.Dir = filepath.Join(os.Getenv("HOME"), "source/premises-mcmanager")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		log.Println(err)
		return false
	}
	s.pid = cmd.Process.Pid
	s.cfg.ServerAddr = "localhost"

	go func() {
		if err := cmd.Wait(); err != nil {
			log.Println(err)
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
		log.Println(err)
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
