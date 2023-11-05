package web

import "github.com/go-webauthn/webauthn/protocol"

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
	ErrPasskeyVerify                         // 10
	ErrPasskeyDup                            // 11
	ErrRequiresAuth                          // 12
	ErrRunnerPrepare                         // 13
	ErrRunnerStop                            // 14
	ErrDNS                                   // 15
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

type Passkey struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type UpdatePassword struct {
	Password    string `json:"password"`
	NewPassword string `json:"newPassword"`
}

type CredentialNameAndCreationResponse struct {
	Name string                              `json:"name"`
	Ccr  protocol.CredentialCreationResponse `json:"credentialCreationResponse"`
}

type EventCode int

const (
	EvStopped      EventCode = iota + 100 // 100
	EvCreateRunner                        // 101
	EvWaitConn                            // 102
	EvConnLost                            // 103
	EvStopRunner                          // 104
)

type PageCode int

const (
	PageLaunch  PageCode = iota + 1 // 1
	PageLoading                     // 2
	PageRunning                     // 3
)

type StandardMessage struct {
	EventCode EventCode `json:"eventCode"`
	Progress  int       `json:"progress"`
	PageCode  PageCode  `json:"pageCode"`
}

type ErrorMessage struct {
	ErrorCode ErrorCode `json:"errorCode"`
}

type SysstatMessage struct {
	CPUUsage float64 `json:"cpuUsage"`
}
