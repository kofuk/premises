package entity

type SessionData struct {
	LoggedIn bool `json:"loggedIn"`
	UserName string `json:"userName"`
}
