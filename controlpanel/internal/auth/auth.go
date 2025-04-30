package auth

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/kofuk/premises/controlpanel/internal/kvs"
)

var ErrNoAuthorization = errors.New("not a bearer header")

const headerAuthorization = "Authorization"

type Scope string

const (
	NoScope    Scope = ""
	ScopeAdmin Scope = "admin"
)

func (scope Scope) String() string {
	return string(scope)
}

type AuthService struct {
	kvs    kvs.KeyValueStore
	host   string
	secret string
}

type Token interface {
	HasScope(scope Scope) bool
	UserID() uint
	TokenString() string
}

type internalToken struct {
	*jwt.Token
	id           string
	scopeMap     map[Scope]struct{}
	userID       uint
	signedString string
}

var _ Token = (*internalToken)(nil)

func (t *internalToken) HasScope(scope Scope) bool {
	if scope == NoScope {
		return true
	}

	_, isAdminScope := t.scopeMap[ScopeAdmin]
	return isAdminScope
}

func (t *internalToken) UserID() uint {
	return t.userID
}

func (t *internalToken) TokenString() string {
	return t.signedString
}

func New(kvs kvs.KeyValueStore, host, secret string) *AuthService {
	return &AuthService{
		kvs:    kvs,
		host:   host,
		secret: secret,
	}
}

func serializeScopes(scopes []Scope) string {
	var scopeIDs []string
	for _, scope := range scopes {
		scopeIDs = append(scopeIDs, scope.String())
	}
	return strings.Join(scopeIDs, ",")
}

func (a *AuthService) CreateToken(ctx context.Context, userID uint, scopes []Scope) (Token, error) {
	currentTime := time.Now()
	id := uuid.NewString()
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.RegisteredClaims{
		Issuer:    a.host,
		Subject:   serializeScopes(scopes),
		Audience:  jwt.ClaimStrings{strconv.Itoa(int(userID))},
		ExpiresAt: jwt.NewNumericDate(currentTime.Add(30 * 24 * time.Hour)),
		NotBefore: jwt.NewNumericDate(currentTime),
		IssuedAt:  jwt.NewNumericDate(currentTime),
		ID:        id,
	})

	scopeMap := make(map[Scope]struct{})
	for _, scope := range scopes {
		scopeMap[scope] = struct{}{}
	}

	signedString, err := token.SignedString([]byte(a.secret))
	if err != nil {
		return nil, err
	}

	return &internalToken{
		Token:        token,
		id:           id,
		scopeMap:     scopeMap,
		userID:       userID,
		signedString: signedString,
	}, nil
}

func (a *AuthService) Get(ctx context.Context, tokenString string) (Token, error) {
	var claim jwt.RegisteredClaims
	token, err := jwt.ParseWithClaims(tokenString, &claim, func(token *jwt.Token) (any, error) {
		return []byte(a.secret), nil
	})
	if err != nil {
		return nil, err
	}

	if a.kvs.Get(ctx, "blocked-token:"+claim.ID, nil) == nil {
		return nil, errors.New("revoked")
	}

	if len(claim.Audience) == 0 {
		return nil, errors.New("missing audience")
	}

	userID, err := strconv.Atoi(claim.Audience[0])
	if err != nil {
		return nil, errors.New("invalid audience")
	}

	scopeMap := make(map[Scope]struct{})
	scopes := strings.Split(claim.Subject, ",")
	for _, scope := range scopes {
		scopeMap[Scope(scope)] = struct{}{}
	}

	return &internalToken{
		Token:        token,
		id:           claim.ID,
		scopeMap:     scopeMap,
		userID:       uint(userID),
		signedString: tokenString,
	}, nil
}

func (a *AuthService) GetFromRequest(ctx context.Context, r *http.Request) (Token, error) {
	header := r.Header.Get(headerAuthorization)
	if header == "" {
		return nil, ErrNoAuthorization
	}

	if !strings.HasPrefix(header, "Bearer ") {
		return nil, ErrNoAuthorization
	}

	return a.Get(ctx, strings.TrimPrefix(header, "Bearer "))
}

func (a *AuthService) RevokeToken(ctx context.Context, tokenString string) error {
	token, err := a.Get(ctx, tokenString)
	if err != nil {
		return err
	}

	if token, ok := token.(*internalToken); ok {
		return a.kvs.Set(ctx, "blocked-token:"+token.id, "", 30*24*time.Hour)
	}
	return nil
}
