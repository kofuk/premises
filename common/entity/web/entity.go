package web

import (
	"github.com/kofuk/premises/common/entity"
)

type ErrorResponse struct {
	Success   bool             `json:"success"`
	ErrorCode entity.ErrorCode `json:"errorCode"`
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

type PendingConfig struct {
	ID            string `json:"id"`
	MachineType   string `json:"machineType"`
	ServerVersion string `json:"serverVersion"`
	GuessVersion  bool   `json:"guessServerVersion"`
	WorldSource   string `json:"worldSource"`
	WorldName     string `json:"worldName"`
	BackupGen     string `json:"backupGen"`
	LevelType     string `json:"levelType"`
	Seed          string `json:"seed"`
}

type LaunchReq struct {
	ID string `json:"id"`
}

type CreateConfigResp struct {
	ID string `json:"id"`
}

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
