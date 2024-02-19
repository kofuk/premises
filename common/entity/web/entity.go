package web

import (
	"github.com/kofuk/premises/common/entity"
)

type ErrorCode int

const (
	ErrBadRequest       ErrorCode = iota + 1 // 1
	ErrInternal                              // 2
	ErrCredential                            // 3
	ErrServerRunning                         // 4
	ErrServerNotRunning                      // 5
	ErrRemote                                // 6
	ErrInvalidConfig                         // 7
	ErrPasswordRule                          // 8
	ErrDupUserName                           // 9
	ErrRequiresAuth                          // 10
	ErrBackup                                // 11
)

const (
	InfoErrRunnerPrepare entity.InfoCode = iota + 100 // 100
	InfoErrRunnerStop                                 // 101
	InfoErrDNS                                        // 102
)

type ErrorResponse struct {
	Success   bool      `json:"success"`
	ErrorCode ErrorCode `json:"errorCode"`
}

type SuccessfulResponse[T any] struct {
	Success bool `json:"success"`
	Data    T    `json:"data"`
}

type SessionState struct {
	NeedsChangePassword bool `json:"needsChangePassword"`
}

type SessionData struct {
	LoggedIn bool   `json:"loggedIn"`
	UserName string `json:"userName"`
}

type MCVersion struct {
	Name        string `json:"name"`
	IsStable    bool   `json:"isStable"`
	Channel     string `json:"channel"`
	ReleaseDate string `json:"releaseDate"`
}

type PasswordCredential struct {
	UserName string `json:"userName"`
	Password string `json:"password"`
}

type BackupGeneration struct {
	Gen       string `json:"gen"`
	ID        string `json:"id"`
	Timestamp int    `json:"timestamp"`
}

type WorldBackup struct {
	WorldName   string             `json:"worldName"`
	Generations []BackupGeneration `json:"generations"`
}

type UpdatePassword struct {
	Password    string `json:"password"`
	NewPassword string `json:"newPassword"`
}

type SystemInfo struct {
	PremisesVersion string  `json:"premisesVersion"`
	HostOS          string  `json:"hostOs"`
	IPAddress       *string `json:"ipAddr"`
}

type WorldInfo struct {
	Version   string `json:"version"`
	WorldName string `json:"worldName"`
	Seed      string `json:"seed"`
}

const (
	EvStopped      entity.EventCode = 100
	EvCreateRunner entity.EventCode = 101
	EvWaitConn     entity.EventCode = 102
	EvConnLost     entity.EventCode = 103
	EvStopRunner   entity.EventCode = 104
	EvManualSetup  entity.EventCode = 105
)

type PageCode int

const (
	PageLaunch      PageCode = 1
	PageLoading     PageCode = 2
	PageRunning     PageCode = 3
	PageManualSetup PageCode = 4
)

type StandardMessage struct {
	EventCode entity.EventCode `json:"eventCode"`
	PageCode  PageCode         `json:"pageCode"`
	Extra     struct {
		Progress int    `json:"progress"`
		TextData string `json:"textData"`
	} `json:"extra"`
}

type InfoMessage struct {
	InfoCode entity.InfoCode `json:"infoCode"`
	IsError  bool            `json:"isError"`
}

type SysstatMessage struct {
	CPUUsage float64 `json:"cpuUsage"`
	Time     int64   `json:"time"`
}

type SnapshotConfiguration struct {
	Slot int `json:"slot"`
}
