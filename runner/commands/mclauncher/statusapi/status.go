package statusapi

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"

	"github.com/kofuk/premises/common/entity/runner"
	"github.com/kofuk/premises/runner/commands/mclauncher/fs"
	"github.com/kofuk/premises/runner/commands/mclauncher/gamesrv"
	"github.com/kofuk/premises/runner/commands/privileged"
	"github.com/kofuk/premises/runner/exterior"
	"github.com/kofuk/premises/runner/systemutil"
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

func LaunchStatusServer(config *runner.Config, srv *gamesrv.ServerInstance) {
	handleStop := func() {
		srv.ShouldStop = true
		srv.Stop()
	}
	handleSnapshot := func() {
		if err := srv.SaveAll(); err != nil {
			slog.Error("Failed to run save-all", slog.Any("error", err))
			return
		}

		_, err := requestQuickSnapshot()
		if err != nil {
			slog.Error("Failed to create snapshot", slog.Any("error", err))

			if err := exterior.SendMessage("serverStatus", runner.Event{
				Type: runner.EventInfo,
				Info: &runner.InfoExtra{
					InfoCode: runner.InfoSnapshotError,
					IsError:  true,
				},
			}); err != nil {
				slog.Error("Unable to write send message", slog.Any("error", err))
			}

			return
		}

		if err := exterior.SendMessage("serverStatus", runner.Event{
			Type: runner.EventInfo,
			Info: &runner.InfoExtra{
				InfoCode: runner.InfoSnapshotDone,
				IsError:  false,
			},
		}); err != nil {
			slog.Error("Unable to write send message", slog.Any("error", err))
		}
	}
	handleUndo := func() {
		go func() {
			if _, err := os.Stat(filepath.Join(fs.LocateWorldData("ss@quick0/world"))); err != nil {
				if err := exterior.SendMessage("serverStatus", runner.Event{
					Type: runner.EventInfo,
					Info: &runner.InfoExtra{
						InfoCode: runner.InfoNoSnapshot,
						IsError:  true,
					},
				}); err != nil {
					slog.Error("Unable to write send message", slog.Any("error", err))
				}
				return
			}

			if err := srv.QuickUndo(); err != nil {
				slog.Error("Unable to quick-undo", slog.Any("error", err))
			}
		}()
	}
	handleReconfigure := func(config *runner.Config) {
		data, err := json.Marshal(&config)
		if err != nil {
			slog.Error("Failed to stringify request", slog.Any("error", err))
			return
		}

		if err := os.WriteFile(fs.LocateDataFile("config.json"), data, 0644); err != nil {
			slog.Error("Failed to write server config", slog.Any("error", err))
			return
		}

		srv.RestartRequested = true
		srv.Stop()
	}

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			slog.Error("Unable to read request body", slog.Any("error", err))
			return
		}

		var action runner.Action
		if err := json.Unmarshal(body, &action); err != nil {
			slog.Error("Unable to unmarshal action", slog.Any("error", err))
			return
		}

		switch action.Type {
		case runner.ActionStop:
			handleStop()
			break

		case runner.ActionSnapshot:
			handleSnapshot()
			break

		case runner.ActionUndo:
			handleUndo()
			break

		case runner.ActionReconfigure:
			handleReconfigure(action.Config)
			break
		}
	})

	// TODO: Send this information to control panel
	http.HandleFunc("/systeminfo", func(w http.ResponseWriter, r *http.Request) {
		systemInfo := systemutil.GetSystemVersion()
		data, err := json.Marshal(systemInfo)
		if err != nil {
			slog.Error("Failed to unmarshal system info", slog.Any("error", err), slog.String("endpoint", "/systeminfo"))
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		w.Write(data)
	})

	// TODO: Send this information to control panel
	http.HandleFunc("/worldinfo", func(w http.ResponseWriter, r *http.Request) {
		if !srv.IsServerInitialized {
			slog.Info("Server is not started. Abort")
			w.WriteHeader(http.StatusTooEarly)
			return
		}

		worldInfo, err := GetWorldInfo(config, srv)
		if err != nil {
			slog.Error("Failed to retrieve world info", slog.Any("error", err), slog.String("endpoint", "/worldinfo"))
			return
		}
		data, err := json.Marshal(worldInfo)
		if err != nil {
			slog.Error("Failed to marshal world info", slog.Any("error", err), slog.String("endpoint", "/worldinfo"))
			return
		}

		w.Write(data)
	})

	slog.Info("Launching status server...")
	if err := http.ListenAndServe("127.0.0.1:9000", nil); err != nil {
		slog.Error("Error listening on :9000", slog.Any("error", err))
		os.Exit(1)
	}
}
