package auth

import (
	"context"
	"encoding/base32"
	"time"

	"github.com/gorilla/securecookie"
	"github.com/kofuk/premises/controlpanel/internal/kvs"
)

type Scope string

const (
	NoScope    Scope = ""
	AdminScope Scope = "admin"
)

type AuthService struct {
	kvs kvs.KeyValueStore
}

type Token struct {
	Token     string    `json:"token"`
	UserID    uint      `json:"userID"`
	Scopes    []Scope   `json:"scopes"`
	CreatedAt time.Time `json:"createdAt"`
}

func New(kvs kvs.KeyValueStore) *AuthService {
	return &AuthService{
		kvs: kvs,
	}
}

func (a *AuthService) CreateToken(ctx context.Context, userID uint, scopes []Scope) (*Token, error) {
	token := &Token{
		Token:     base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(securecookie.GenerateRandomKey(32)),
		UserID:    userID,
		Scopes:    scopes,
		CreatedAt: time.Now(),
	}

	if err := a.kvs.Set(ctx, "token:"+token.Token, token, 30*24*time.Hour); err != nil {
		return nil, err
	}

	return token, nil
}

func (a *AuthService) Get(ctx context.Context, token string) (*Token, error) {
	var t Token
	if err := a.kvs.Get(ctx, "token:"+token, &t); err != nil {
		return nil, err
	}

	return &t, nil
}

func (a *AuthService) RevokeToken(ctx context.Context, token string) error {
	return a.kvs.Del(ctx, "token:"+token)
}

func (a *AuthService) IsGranted(ctx context.Context, token string, scope Scope) (bool, error) {
	var t Token
	if err := a.kvs.Get(ctx, "token:"+token, &t); err != nil {
		return false, err
	}

	if scope == NoScope {
		return true, nil
	}

	for _, s := range t.Scopes {
		if s == scope {
			return true, nil
		}
	}

	return false, nil
}
