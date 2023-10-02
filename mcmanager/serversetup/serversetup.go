package serversetup

import (
	"crypto/tls"
	"encoding/json"
	"net/http"
	"os"
	"os/exec"
	"os/user"
	"time"

	"github.com/kofuk/premises/mcmanager/statusapi"
	"github.com/kofuk/premises/mcmanager/systemutil"
	log "github.com/sirupsen/logrus"
)

var requiredProgs = []string{
	"mkfs.btrfs",
	"java",
	"ufw",
	"unzip",
}

type ServerSetup struct {
	statusServer *http.Server
}

func (self *ServerSetup) launchStatus() {
	if self.statusServer != nil {
		return
	}

	http.HandleFunc("/monitor", func(w http.ResponseWriter, r *http.Request) {
		writeJson := func(w http.ResponseWriter, data *statusapi.StatusData) error {
			json, err := json.Marshal(data)
			if err != nil {
				return err
			}
			json = append(json, '\n')
			if _, err := w.Write(json); err != nil {
				return err
			}

			w.(http.Flusher).Flush()

			return nil
		}

		writeJson(w, &statusapi.StatusData{
			Type:     statusapi.StatusTypeLegacyEvent,
			Status:   "サーバを初期化しています…",
			Shutdown: false,
			HasError: false,
		})

		<-make(chan struct{})
	})

	tlsCfg := &tls.Config{
		MinVersion:               tls.VersionTLS13,
		CurvePreferences:         []tls.CurveID{tls.CurveP521, tls.CurveP384, tls.CurveP256},
		PreferServerCipherSuites: true,
		CipherSuites: []uint16{
			tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA,
			tls.TLS_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_RSA_WITH_AES_256_CBC_SHA,
		},
	}

	self.statusServer = &http.Server{
		Addr:         "0.0.0.0:8521",
		TLSConfig:    tlsCfg,
		TLSNextProto: make(map[string]func(*http.Server, *tls.Conn, http.Handler), 0),
		ReadTimeout:  5 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	log.Info("Launching status server...")
	if err := self.statusServer.ListenAndServeTLS("/opt/premises/server.crt", "/opt/premises/server.key"); err != nil {
		log.Println(err)
	}
}

func isServerInitialized() bool {
	for _, prog := range requiredProgs {
		_, err := exec.LookPath(prog)
		if err != nil {
			log.Println("Required executable not found:", prog)
			return false
		}
	}

	if _, err := os.Stat("/opt/premises/gamedata"); os.IsNotExist(err) {
		return false
	}

	return true
}

func isDevEnv() bool {
	_, err := os.Stat("/.dockerenv")
	return err == nil
}

func (self *ServerSetup) initializeServer() {
	go self.launchStatus()

	log.Println("Updating package indices")
	systemutil.AptGet("update", "-y")

	log.Println("Installing packages")
	systemutil.AptGet("install", "-y", "btrfs-progs", "openjdk-17-jre-headless", "ufw", "unzip")

	if _, err := user.LookupId("1000"); err != nil {
		log.Println("Adding user")
		systemutil.Cmd("useradd", []string{"-U", "-s", "/bin/bash", "-u", "1000", "premises"}, nil)
	}

	if !isDevEnv() {
		log.Println("Enabling ufw")
		systemutil.Cmd("systemctl", []string{"enable", "--now", "ufw.service"}, nil)
		systemutil.Cmd("ufw", []string{"enable"}, nil)

		log.Println("Adding ufw rules")
		systemutil.Cmd("ufw", []string{"allow", "25565/tcp"}, nil)
		systemutil.Cmd("ufw", []string{"allow", "8521/tcp"}, nil)
	}

	log.Println("Creating data directories")
	os.MkdirAll("/opt/premises/servers.d/../gamedata", 0755)

	if _, err := os.Stat("/opt/premises/gamedata.img"); os.IsNotExist(err) {
		log.Println("Creating image file to save game data")
		size := "8G"
		if isDevEnv() {
			size = "1G"
		}
		systemutil.Cmd("fallocate", []string{"-l", size, "/opt/premises/gamedata.img"}, nil)

		log.Println("Creating filesystem for gamedata.img")
		systemutil.Cmd("mkfs.btrfs", []string{"/opt/premises/gamedata.img"}, nil)
	}
}

func (self ServerSetup) Run() {
	if !isServerInitialized() {
		log.Println("Server seems not to be initialized. Will run full initialization")
		self.initializeServer()
	}

	log.Println("Mounting gamedata.img")
	systemutil.Cmd("mount", []string{"/opt/premises/gamedata.img", "/opt/premises/gamedata"}, nil)

	log.Println("Ensure data directory owned by execution user")
	systemutil.Cmd("chown", []string{"-R", "1000:1000", "/opt/premises"}, nil)

	if self.statusServer != nil {
		self.statusServer.Close()
	}
}
