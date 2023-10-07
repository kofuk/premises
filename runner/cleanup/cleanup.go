package cleanup

import (
	"crypto/tls"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"syscall"
	"time"

	"github.com/kofuk/premises/runner/statusapi"
	"github.com/kofuk/premises/runner/systemutil"
	log "github.com/sirupsen/logrus"
)

var shutdown = false

func launchStatus() {
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

		ticker := time.NewTicker(1 * time.Second)

		for {
			<-ticker.C

			if err := writeJson(w, &statusapi.StatusData{
				Type:     statusapi.StatusTypeLegacyEvent,
				Status:   "終了準備しています…",
				Shutdown: shutdown,
				HasError: false,
			}); err != nil {
				break
			}
		}

		ticker.Stop()
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

	statusServer := &http.Server{
		Addr:         "0.0.0.0:8521",
		TLSConfig:    tlsCfg,
		TLSNextProto: make(map[string]func(*http.Server, *tls.Conn, http.Handler), 0),
		ReadTimeout:  5 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	log.Info("Launching status server...")
	if err := statusServer.ListenAndServeTLS("/opt/premises/server.crt", "/opt/premises/server.key"); err != nil {
		log.Println(err)
	}
}

func removeFilesIgnoreError(paths ...string) {
	for _, path := range paths {
		if err := os.Remove(path); err != nil {
			log.WithError(err).WithField("path", path).Info("Failed to clean up file")
		}
	}
}

func removeSnapshots() {
	dirent, err := os.ReadDir("/opt/premises/gamedata")
	if err != nil {
		log.WithError(err).Error("Error reading data dir")
		return
	}

	args := []string{"subvolume", "delete", "--commit-after"}
	for _, ent := range dirent {
		if ent.Name()[:3] == "ss@" {
			args = append(args, filepath.Join("/opt/premises/gamedata", ent.Name()))
		}
	}

	if err := systemutil.Cmd("btrfs", args, nil); err != nil {
		log.WithError(err).Info("Failed to remove snapshots")
	}
}

func unmountData() {
	if err := syscall.Unmount("/opt/premises/gamedata", 0); err != nil {
		log.WithError(err).Error("Error unmounting data dir")
	}
}

func CleanUp() {
	go launchStatus()

	// XXX
	time.Sleep(5 * time.Second)

	log.Info("Removing config files...")
	removeFilesIgnoreError(
		"/opt/premises/server.key",
		"/opt/premises/server.crt",
		"/opt/premises/config.json",
		"/userdata",
		"/userdata_decoded.sh",
	)

	log.Info("Removing snaphots...")
	removeSnapshots()

	log.Info("Unmounting data dir...")
	unmountData()

	shutdown = true

	<-make(chan struct{})
}
