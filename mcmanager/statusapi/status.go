package statusapi

import (
	"bytes"
	"crypto/subtle"
	"crypto/tls"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/kofuk/premises/mcmanager/backup"
	"github.com/kofuk/premises/mcmanager/config"
	"github.com/kofuk/premises/mcmanager/gamesrv"
	"github.com/kofuk/premises/mcmanager/privileged"
	"github.com/kofuk/premises/mcmanager/systemutil"
	log "github.com/sirupsen/logrus"
)

const (
	StatusTypeLegacyEvent = "legacyEvent"
	StatusTypeSystemStat  = "systemStat"
)

type StatusType string

type StatusData struct {
	Type     StatusType `json:"type"`
	Status   string     `json:"status"`
	Shutdown bool       `json:"shutdown"`
	HasError bool       `json:"hasError"`
	CPUUsage float64    `json:"cpuUsage"`
}

type createSnapshotResp struct {
	Version int                     `json:"version"`
	Success bool                    `json:"success"`
	Message string                  `json:"message"`
	Result  privileged.SnapshotInfo `json:"result"`
}

func uploadSnapshot(ctx *config.PMCMContext, ssi *privileged.SnapshotInfo) {
	options := backup.UploadOptions{
		TmpFileName: "ss@" + ssi.ID + ".tar.zst",
		SourceDir:   ssi.Path,
	}

	ctx.NotifyStatus(ctx.L("snapshot.processing"))
	if err := backup.PrepareUploadData(ctx, options); err != nil {
		ctx.NotifyStatus(ctx.L("snapshot.process.error"))
		goto out
	}

	ctx.NotifyStatus(ctx.L("world.uploading"))
	if err := backup.UploadWorldData(ctx, options); err != nil {
		ctx.NotifyStatus(ctx.L("world.upload.error"))
		goto out
	}
out:

	if err := requestDeleteSnapshot(ssi); err != nil {
		ctx.NotifyStatus(ctx.L("snapshot.clean.error"))
	} else {
		log.Info("Successfully cleaned snapshot")
	}

	ctx.NotifyStatus(ctx.L("game.running"))
}

func requestSnapshot() (*privileged.SnapshotInfo, error) {
	reqMsg := &privileged.RequestMsg{
		Version: 1,
		Func:    "snapshots/create",
	}
	reqData, err := json.Marshal(reqMsg)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest(http.MethodPost, "http://localhost:8522", bytes.NewBuffer(reqData))
	if err != nil {
		return nil, err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}

	respData, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var respMsg createSnapshotResp
	if err := json.Unmarshal(respData, &respMsg); err != nil {
		return nil, err
	}

	if respMsg.Version != 1 {
		return nil, errors.New("Unsupported version")
	}

	if !respMsg.Success {
		return nil, errors.New(respMsg.Message)
	}

	return &respMsg.Result, nil
}

func requestQuickSnapshot() (*privileged.SnapshotInfo, error) {
	reqMsg := &privileged.RequestMsg{
		Version: 1,
		Func:    "quicksnapshots/create",
	}
	reqData, err := json.Marshal(reqMsg)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest(http.MethodPost, "http://localhost:8522", bytes.NewBuffer(reqData))
	if err != nil {
		return nil, err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}

	respData, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var respMsg createSnapshotResp
	if err := json.Unmarshal(respData, &respMsg); err != nil {
		return nil, err
	}

	if respMsg.Version != 1 {
		return nil, errors.New("Unsupported version")
	}

	if !respMsg.Success {
		return nil, errors.New(respMsg.Message)
	}

	return &respMsg.Result, nil
}

func requestDeleteSnapshot(ssi *privileged.SnapshotInfo) error {
	reqMsg := &privileged.RequestMsg{
		Version: 1,
		Func:    "snapshots/delete",
		Args: []string{
			ssi.ID,
		},
	}
	reqData, err := json.Marshal(reqMsg)
	if err != nil {
		return err
	}
	req, err := http.NewRequest(http.MethodPost, "http://localhost:8522", bytes.NewBuffer(reqData))
	if err != nil {
		return err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}

	respData, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	var respMsg createSnapshotResp
	if err := json.Unmarshal(respData, &respMsg); err != nil {
		return err
	}

	if respMsg.Version != 1 {
		return errors.New("Unsupported version")
	}

	if !respMsg.Success {
		return errors.New(respMsg.Message)
	}

	return nil
}

func LaunchStatusServer(ctx *config.PMCMContext, srv *gamesrv.ServerInstance) {
	http.HandleFunc("/monitor", func(w http.ResponseWriter, r *http.Request) {
		authKey := r.Header.Get("X-Auth-Key")
		if subtle.ConstantTimeCompare([]byte(authKey), []byte(ctx.Cfg.AuthKey)) == 0 {
			log.WithField("endpoint", "/monitor").Error("Invalid auth key")
			w.WriteHeader(http.StatusForbidden)
			return
		}

		writeJson := func(w http.ResponseWriter, data *StatusData) error {
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

		if err := writeJson(w, &StatusData{
			Type:     StatusTypeLegacyEvent,
			Status:   ctx.LastStatus,
			Shutdown: srv.IsServerFinished,
			HasError: srv.StartupFailed,
		}); err != nil {
			log.WithError(err).Error("Failed to write status")
			return
		}
		if srv.IsServerFinished {
			// Connected to shutdown server.
			// Notify the state and close.
			return
		}

		statusChannel := make(chan string)
		defer close(statusChannel)
		ctx.AddStatusChannel(statusChannel)
		defer ctx.RemoveStatusChannel(statusChannel)

		cpuStat, err := systemutil.NewCPUUsage()
		if err != nil {
			log.WithError(err).Error("Failed to initialize CPU usage")
		}

		var tickerChan <-chan time.Time
		if cpuStat != nil {
			ticker := time.NewTicker(3 * time.Second)
			tickerChan = ticker.C
			defer ticker.Stop()
		}

	L:
		for {
			select {
			case status, ok := <-statusChannel:
				if !ok {
					break L
				}
				if err := writeJson(w, &StatusData{
					Type:     StatusTypeLegacyEvent,
					Status:   status,
					Shutdown: srv.IsServerFinished,
					HasError: srv.StartupFailed,
				}); err != nil {
					log.WithError(err).Error("Failed to write data to connection")
					break L
				}

			case <-tickerChan:
				cpuUsage, err := cpuStat.Percent()
				if err != nil {
					log.WithError(err).Error("Failed to retrieve CPU usage")
					continue
				}

				if err := writeJson(w, &StatusData{
					Type:     StatusTypeSystemStat,
					Shutdown: srv.IsServerFinished,
					HasError: srv.StartupFailed,
					CPUUsage: cpuUsage,
				}); err != nil {
					log.WithError(err).Error("Failed to write data to connection")
					break L
				}

			case <-r.Context().Done():
				break L
			}
		}
	})

	http.HandleFunc("/newconfig", func(w http.ResponseWriter, r *http.Request) {
		authKey := r.Header.Get("X-Auth-Key")
		if subtle.ConstantTimeCompare([]byte(authKey), []byte(ctx.Cfg.AuthKey)) == 0 {
			log.WithField("endpoint", "/newconfig").Error("Invalid auth key")
			w.WriteHeader(http.StatusForbidden)
			return
		}

		var config config.Config
		if err := json.NewDecoder(r.Body).Decode(&config); err != nil {
			log.WithError(err).Error("Failed to parse request JSON")
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		data, err := json.Marshal(&config)
		if err != nil {
			log.WithError(err).Error("Failed to stringify request")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		if err := os.WriteFile(ctx.LocateDataFile("config.json"), data, 0644); err != nil {
			log.WithError(err).Error("Failed to write server config")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		srv.RestartRequested = true
		srv.Stop()
	})

	http.HandleFunc("/snapshot", func(w http.ResponseWriter, r *http.Request) {
		authKey := r.Header.Get("X-Auth-Key")
		if subtle.ConstantTimeCompare([]byte(authKey), []byte(ctx.Cfg.AuthKey)) == 0 {
			log.WithField("endpoint", "/snapshot").Error("Invalid auth key")
			w.WriteHeader(http.StatusForbidden)
			return
		}

		if err := srv.SaveAll(); err != nil {
			log.WithError(err).Error("Failed to run save-all")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		ssi, err := requestSnapshot()
		if err != nil {
			log.WithError(err).Error("Failed to create snapshot")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusCreated)

		go uploadSnapshot(ctx, ssi)
	})

	http.HandleFunc("/quickss", func(w http.ResponseWriter, r *http.Request) {
		authKey := r.Header.Get("X-Auth-Key")
		if subtle.ConstantTimeCompare([]byte(authKey), []byte(ctx.Cfg.AuthKey)) == 0 {
			log.WithField("endpoint", "/quickss").Error("Invalid auth key")
			w.WriteHeader(http.StatusForbidden)
			return
		}

		if err := srv.SaveAll(); err != nil {
			log.WithError(err).Error("Failed to run save-all")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		_, err := requestQuickSnapshot()
		if err != nil {
			log.WithError(err).Error("Failed to create snapshot")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		srv.SendChat("スナップショットを取得しました！")

		w.WriteHeader(http.StatusCreated)
	})

	http.HandleFunc("/quickundo", func(w http.ResponseWriter, r *http.Request) {
		authKey := r.Header.Get("X-Auth-Key")
		if subtle.ConstantTimeCompare([]byte(authKey), []byte(ctx.Cfg.AuthKey)) == 0 {
			log.WithField("endpoint", "/quickss").Error("Invalid auth key")
			w.WriteHeader(http.StatusForbidden)
			return
		}

		go func() {
			srv.SendChat("3秒後にサーバを再起動します")
			time.Sleep(time.Second)
			srv.SendChat("2…")
			time.Sleep(time.Second)
			srv.SendChat("1…")
			time.Sleep(time.Second)
			srv.QuickUndo()
		}()

		w.WriteHeader(http.StatusCreated)
	})

	http.HandleFunc("/stop", func(w http.ResponseWriter, r *http.Request) {
		authKey := r.Header.Get("X-Auth-Key")
		if subtle.ConstantTimeCompare([]byte(authKey), []byte(ctx.Cfg.AuthKey)) == 0 {
			log.WithField("endpoint", "/stop").Error("Invalid auth key")
			w.WriteHeader(http.StatusForbidden)
			return
		}

		srv.ShouldStop = true
		srv.Stop()
	})

	http.HandleFunc("/systeminfo", func(w http.ResponseWriter, r *http.Request) {
		authKey := r.Header.Get("X-Auth-Key")
		if subtle.ConstantTimeCompare([]byte(authKey), []byte(ctx.Cfg.AuthKey)) == 0 {
			log.WithField("endpoint", "/systeminfo").Error("Invalid auth key")
			w.WriteHeader(http.StatusForbidden)
			return
		}

		systemInfo := systemutil.GetSystemVersion()
		data, err := json.Marshal(systemInfo)
		if err != nil {
			log.WithError(err).WithField("endpoint", "/systeminfo").Error("Failed to unmarshal system info")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		w.Write(data)
	})

	http.HandleFunc("/worldinfo", func(w http.ResponseWriter, r *http.Request) {
		authKey := r.Header.Get("X-Auth-Key")
		if subtle.ConstantTimeCompare([]byte(authKey), []byte(ctx.Cfg.AuthKey)) == 0 {
			log.WithField("endpoint", "/systeminfo").Error("Invalid auth key")
			w.WriteHeader(http.StatusForbidden)
			return
		}

		if !srv.IsServerInitialized {
			log.Info("Server is not started. Abort")
			w.WriteHeader(http.StatusTooEarly)
			return
		}

		worldInfo, err := GetWorldInfo(ctx, srv)
		if err != nil {
			log.WithError(err).WithField("endpoint", "/worldinfo").Error("Failed to retrieve world info")
			return
		}
		data, err := json.Marshal(worldInfo)
		if err != nil {
			log.WithError(err).WithField("endpoint", "/worldinfo").Error("Failed to marshal world info")
			return
		}

		w.Write(data)
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
	log.Fatal(server.ListenAndServeTLS(ctx.LocateDataFile("server.crt"), ctx.LocateDataFile("server.key")))
}
