package serversetup

import (
	"crypto/tls"
	"encoding/json"
	"net/http"
	"os"
	"os/exec"
	"os/user"
	"syscall"
	"time"

	"github.com/kofuk/premises/mcmanager/statusapi"
	log "github.com/sirupsen/logrus"
)

var requiredProgs = []string{
	"mkfs.btrfs",
	"java",
	"utw",
	"unzip",
}

type ServerSetup struct {
	statusLaunched bool
}

func (self *ServerSetup) launchStatus() {
	if self.statusLaunched {
		return
	}
	self.statusLaunched = true

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

	server := &http.Server{
		Addr:         "0.0.0.0:8521",
		TLSConfig:    tlsCfg,
		TLSNextProto: make(map[string]func(*http.Server, *tls.Conn, http.Handler), 0),
		ReadTimeout:  5 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	log.Info("Launching status server...")
	log.Fatal(server.ListenAndServeTLS("/opt/premises/server.crt", "/opt/premises/server.key"))

}

func isServerInitialized() bool {
	for _, prog := range requiredProgs {
		_, err := exec.LookPath(prog)
		if err != nil {
			return false
		}
	}
	return true
}

func (self *ServerSetup) initializeServer() {
	go self.launchStatus()

	log.Println("Updating package indices")
	cmd("apt-get", []string{
		"update", "-y",
	}, []string{"DEBIAN_FRONTEND=noninteractive"})

	log.Println("Installing packages")
	cmd("apt-get", []string{
		"install", "-y", "btrfs-progs", "openjdk-17-jre-headless", "ufw", "unzip",
	}, []string{"DEBIAN_FRONTEND=noninteractive"})

	if _, err := user.LookupId("1000"); err != nil {
		log.Println("Adding user")
		cmd("useradd", []string{
			"-U", "-s", "/bin/bash", "-u", "1000", "premises",
		}, []string{"DEBIAN_FRONTEND=noninteractive"})
	}

	if hasSystemd() {
		log.Println("Enabling ufw")
		cmd("systemctl", []string{"enable", "--now", "ufw.service"}, nil)
		cmd("ufw", []string{"enable"}, nil)

		log.Println("Adding ufw rules")
		cmd("ufw", []string{"allow", "25565/tcp"}, nil)
		cmd("ufw", []string{"allow", "8521/tcp"}, nil)
	}

	log.Println("Creating data directories")
	os.MkdirAll("/opt/premises/servers.d/../gamedata", 0755)

	if _, err := os.Stat("/opt/premises/gamedata.img"); os.IsNotExist(err) {
		log.Println("Creating image file to save game data")
		file, err := os.Create("/opt/premises/gamedata.img")
		if err != nil {
			log.Println(err)
		} else {
			if err := syscall.Fallocate(int(file.Fd()), 0644, 0, 8*1024*1024*1024); err != nil {
				log.Println(err)
			}
			file.Close()
		}

		log.Println("Creating filesystem for gamedata.img")
		cmd("mkfs.btrfs", []string{"/opt/premises/gamedata.img"}, nil)
	}
}

func cmd(cmdPath string, args []string, envs []string) error {
	cmd := exec.Command(cmdPath, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	for _, env := range envs {
		cmd.Env = append(cmd.Env, env)
	}
	if err := cmd.Run(); err != nil {
		return err
	}
	return nil
}

func hasSystemd() bool {
	_, err := exec.LookPath("systemctl")
	return err == nil
}

func (self ServerSetup) Run() {
	if !isServerInitialized() {
		self.initializeServer()
	}

	log.Println("Mounting gamedata.img")
	cmd("mount", []string{"/opt/premises/gamedata.img", "/opt/premises/gamedata"}, nil)

	log.Println("Ensure data directory owned by execution user")
	cmd("chown", []string{"-R", "1000:1000", "/opt/premises"}, nil)
}
