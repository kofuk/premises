package auth

import (
	"context"
	"encoding/base32"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/securecookie"
	"github.com/kofuk/premises/controlpanel/internal/kvs"
)

var ErrNoAuthorization = errors.New("not a bearer header")

const headerAuthorization = "Authorization"

type Scope string

const (
	NoScope    Scope = ""
	ScopeAdmin Scope = "admin"
)

type AuthService struct {
	kvs kvs.KeyValueStore
}

type Token struct {
	Token     string    `json:"token"`
	UserID    uint      `json:"userID"`
	Scopes    []Scope   `json:"scopes"`
	CreatedAt time.Time `json:"createdAt"`
	scopeMap  map[Scope]struct{}
}

func (t *Token) HasScope(scope Scope) bool {
	if scope == NoScope {
		return true
	}

	_, isAdminScope := t.scopeMap[ScopeAdmin]
	return isAdminScope
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

	t.scopeMap = make(map[Scope]struct{})
	for _, scope := range t.Scopes {
		if scope == NoScope {
			continue
		}
		t.scopeMap[scope] = struct{}{}
	}

	return &t, nil
}

func (a *AuthService) GetFromRequest(ctx context.Context, r *http.Request) (*Token, error) {
	header := r.Header.Get(headerAuthorization)
	if header == "" {
		return nil, ErrNoAuthorization
	}

	if !strings.HasPrefix(header, "Bearer ") {
		return nil, ErrNoAuthorization
	}

	return a.Get(ctx, strings.TrimPrefix(header, "Bearer "))
}

func (a *AuthService) RevokeToken(ctx context.Context, token string) error {
	return a.kvs.Del(ctx, "token:"+token)
}
