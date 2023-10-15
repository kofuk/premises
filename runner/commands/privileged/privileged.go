package privileged

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
)

type RequestMsg struct {
	Version int      `json:"version"`
	Func    string   `json:"func"`
	Args    []string `json:"args"`
}

type ResponseMsg struct {
	Version int         `json:"version"`
	Success bool        `json:"success"`
	Message string      `json:"message,omitempty"`
	Result  interface{} `json:"result,omitempty"`
}

type SnapshotInfo struct {
	ID   string `json:"id"`
	Path string `json:"path"`
}

func takeFsSnapshot(snapshotId string) (*SnapshotInfo, error) {
	if snapshotId == "" {
		id, err := uuid.NewUUID()
		if err != nil {
			return nil, err
		}
		snapshotId = id.String()
	}

	gameDir := "/opt/premises/gamedata"

	var snapshotInfo SnapshotInfo
	snapshotInfo.ID = snapshotId
	snapshotInfo.Path = filepath.Join(gameDir, "ss@"+snapshotId)

	if snapshotId != "" {
		if err := deleteFsSnapshot(snapshotId); err != nil {
			log.WithError(err).Info("Failed to remove old snapshot (doesn't the snapshot exist?)")
		}
	}

	// Create read-only snapshot
	cmd := exec.Command("btrfs", "subvolume", "snapshot", "-r", ".", snapshotInfo.Path)
	cmd.Dir = gameDir
	if err := cmd.Run(); err != nil {
		return nil, err
	}

	return &snapshotInfo, nil
}

func deleteFsSnapshot(id string) error {
	if strings.Contains(id, "/") {
		return errors.New("Invalid snapshot ID")
	}

	gameDir := "/opt/premises/gamedata"

	cmd := exec.Command("btrfs", "subvolume", "delete", "ss@"+id)
	cmd.Dir = gameDir
	if err := cmd.Run(); err != nil {
		return err
	}

	cmd = exec.Command("btrfs", "balance", ".")
	cmd.Dir = gameDir
	if err := cmd.Run(); err != nil {
		return err
	}

	return nil
}

func sendMessage(w http.ResponseWriter, msg interface{}) error {
	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	w.Header().Add("Content-Type", "application/json")
	if _, err := w.Write(data); err != nil {
		return err
	}
	return nil
}

func Run() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		log.Info("Request received")

		body, err := io.ReadAll(r.Body)
		if err != nil {
			log.WithError(err).Error("Failed to read body")
			err := sendMessage(w, &ResponseMsg{
				Version: 1,
				Success: false,
				Message: "Failed to read body",
			})
			if err != nil {
				log.WithError(err).Error("Failed to write body")
			}
			return
		}

		var req RequestMsg
		if err := json.Unmarshal(body, &req); err != nil {
			log.WithError(err).Error("Failed to parse body")
			err := sendMessage(w, &ResponseMsg{
				Version: 1,
				Success: false,
				Message: "failed to parse body",
			})
			if err != nil {
				log.WithError(err).Error("Failed to write body")
			}
			return
		}

		if req.Version != 1 {
			log.WithField("version", req.Version).Error("Unsupported version")
			err := sendMessage(w, &ResponseMsg{
				Version: 1,
				Success: false,
				Message: "Unsupported version",
			})
			if err != nil {
				log.WithError(err).Error("Failed to write body")
			}
			return
		}

		if req.Func == "snapshots/create" {
			ssi, err := takeFsSnapshot("")
			if err != nil {
				log.WithError(err).Error("Failed to take snapshot")

				err := sendMessage(w, &ResponseMsg{
					Version: 1,
					Success: false,
					Message: "Failed to take snapshot",
				})
				if err != nil {
					log.WithError(err).Error("Failed to write body")
					return
				}
				return
			}

			err = sendMessage(w, &ResponseMsg{
				Version: 1,
				Success: true,
				Result:  ssi,
			})
			if err != nil {
				log.WithError(err).Error("Failed to write body")
				return
			}
		} else if req.Func == "snapshots/delete" {
			if len(req.Args) != 1 {
				log.Error("Invalid argument")

				err = sendMessage(w, &ResponseMsg{
					Version: 1,
					Success: true,
					Message: "Invalid argument",
				})
				if err != nil {
					log.WithError(err).Error("Failed to write body")
					return
				}
			}

			if err := deleteFsSnapshot(req.Args[0]); err != nil {
				log.WithError(err).Error("Failed to delete snapshot")

				err = sendMessage(w, &ResponseMsg{
					Version: 1,
					Success: true,
					Message: "Failed to take snapshot",
				})
				if err != nil {
					log.WithError(err).Error("Failed to write body")
					return
				}
				return
			}

			err = sendMessage(w, &ResponseMsg{
				Version: 1,
				Success: true,
			})
			if err != nil {
				log.WithError(err).Error("Failed to write body")
				return
			}
		} else if req.Func == "quicksnapshots/create" {
			ssi, err := takeFsSnapshot("quick0")
			if err != nil {
				log.WithError(err).Error("Failed to take snapshot")

				err := sendMessage(w, &ResponseMsg{
					Version: 1,
					Success: false,
					Message: "Failed to take snapshot",
				})
				if err != nil {
					log.WithError(err).Error("Failed to write body")
					return
				}
				return
			}

			err = sendMessage(w, &ResponseMsg{
				Version: 1,
				Success: true,
				Result:  ssi,
			})
			if err != nil {
				log.WithError(err).Error("Failed to write body")
				return
			}
		} else {
			log.Error("Unknown method")
			err := sendMessage(w, &ResponseMsg{
				Version: 1,
				Success: false,
				Message: "Unknown method",
			})
			if err != nil {
				log.WithError(err).Error("Failed to write body")
			}
		}
	})

	log.Fatal(http.ListenAndServe("localhost:8522", nil))
}
