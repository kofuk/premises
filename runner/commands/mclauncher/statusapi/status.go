package statusapi

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"

	"github.com/kofuk/premises/runner/commands/mclauncher/config"
	"github.com/kofuk/premises/runner/commands/mclauncher/gamesrv"
	"github.com/kofuk/premises/runner/commands/privileged"
	"github.com/kofuk/premises/runner/exterior"
	"github.com/kofuk/premises/runner/exterior/entity"
	"github.com/kofuk/premises/runner/systemutil"
	log "github.com/sirupsen/logrus"
)

type createSnapshotResp struct {
	Version int                     `json:"version"`
	Success bool                    `json:"success"`
	Message string                  `json:"message"`
	Result  privileged.SnapshotInfo `json:"result"`
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
	http.HandleFunc("/newconfig", func(w http.ResponseWriter, r *http.Request) {
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

	http.HandleFunc("/quickss", func(w http.ResponseWriter, r *http.Request) {
		if err := srv.SaveAll(); err != nil {
			log.WithError(err).Error("Failed to run save-all")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		_, err := requestQuickSnapshot()
		if err != nil {
			log.WithError(err).Error("Failed to create snapshot")

			if err := exterior.SendMessage("serverStatus", entity.Event{
				Type: entity.EventInfo,
				Info: &entity.InfoExtra{
					InfoCode:  entity.InfoSnapshotError,
					LegacyMsg: "スナップショットを取得できませんでした",
				},
			}); err != nil {
				log.WithError(err).Error("Unable to write send message")
			}

			return
		}

		if err := exterior.SendMessage("serverStatus", entity.Event{
			Type: entity.EventInfo,
			Info: &entity.InfoExtra{
				InfoCode:  entity.InfoSnapshotDone,
				LegacyMsg: "スナップショットを取得しました！",
			},
		}); err != nil {
			log.WithError(err).Error("Unable to write send message")
		}

		w.WriteHeader(http.StatusCreated)
	})

	http.HandleFunc("/quickundo", func(w http.ResponseWriter, r *http.Request) {
		go srv.QuickUndo()

		w.WriteHeader(http.StatusCreated)
	})

	http.HandleFunc("/stop", func(w http.ResponseWriter, r *http.Request) {
		srv.ShouldStop = true
		srv.Stop()
	})

	http.HandleFunc("/systeminfo", func(w http.ResponseWriter, r *http.Request) {
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

	log.Info("Launching status server...")
	log.Fatal(http.ListenAndServe("127.0.0.1:9000", nil))
}
