package privileged

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
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
			slog.Error("Failed to remove old snapshot (doesn't the snapshot exist?)", slog.Any("error", err))
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

func Run(args []string) {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		slog.Info("Request received")

		body, err := io.ReadAll(r.Body)
		if err != nil {
			slog.Error("Failed to read body", slog.Any("error", err))
			err := sendMessage(w, &ResponseMsg{
				Version: 1,
				Success: false,
				Message: "Failed to read body",
			})
			if err != nil {
				slog.Error("Failed to write body", slog.Any("error", err))
			}
			return
		}

		var req RequestMsg
		if err := json.Unmarshal(body, &req); err != nil {
			slog.Error("Failed to parse body", slog.Any("error", err))
			err := sendMessage(w, &ResponseMsg{
				Version: 1,
				Success: false,
				Message: "failed to parse body",
			})
			if err != nil {
				slog.Error("Failed to write body", slog.Any("error", err))
			}
			return
		}

		if req.Version != 1 {
			slog.Error("Unsupported version", slog.Any("version", req.Version))
			err := sendMessage(w, &ResponseMsg{
				Version: 1,
				Success: false,
				Message: "Unsupported version",
			})
			if err != nil {
				slog.Error("Failed to write body", slog.Any("error", err))
			}
			return
		}

		if req.Func == "quicksnapshots/create" {
			if len(req.Args) == 0 {
				req.Args = append(req.Args, "0")
			}

			ssi, err := takeFsSnapshot(fmt.Sprintf("quick%s", req.Args[0]))
			if err != nil {
				slog.Error("Failed to take snapshot", slog.Any("error", err))

				err := sendMessage(w, &ResponseMsg{
					Version: 1,
					Success: false,
					Message: "Failed to take snapshot",
				})
				if err != nil {
					slog.Error("Failed to write body", slog.Any("error", err))
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
				slog.Error("Failed to write body", slog.Any("error", err))
				return
			}
		} else {
			slog.Error("Unknown method")
			err := sendMessage(w, &ResponseMsg{
				Version: 1,
				Success: false,
				Message: "Unknown method",
			})
			if err != nil {
				slog.Error("Failed to write body", slog.Any("error", err))
			}
		}
	})

	if err := http.ListenAndServe("localhost:8522", nil); err != nil {
		slog.Error("Unable to listen :8522", slog.Any("error", err))
		os.Exit(1)
	}
}
