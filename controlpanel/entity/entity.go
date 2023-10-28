package entity

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
)

type ErrorResponse struct {
	Success   bool      `json:"success"`
	ErrorCode ErrorCode `json:"errorCode"`
	Reason    string    `json:"reason"`
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
