package entity

type ErrorCode int

const (
	ErrBadRequest ErrorCode = iota + 1
	ErrInternal
	ErrCredential
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
