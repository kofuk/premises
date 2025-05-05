package runners

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/kofuk/premises/controlpanel/internal/launcher/server"
	"github.com/kofuk/premises/controlpanel/internal/launcher/server/conoha"
	"github.com/kofuk/premises/controlpanel/internal/runners/repository"
)

var providers = map[string]func(ctx context.Context, data map[string]string) (server.GameServer, error){
	"Conoha": conoha.NewWithProviderSpecificData,
}

type RunnersService struct {
	repository  *repository.RunnerRepository
	tokenIssuer string
	secret      string
}

func NewRunnersService(repository *repository.RunnerRepository, domain, secret string) *RunnersService {
	return &RunnersService{
		repository:  repository,
		tokenIssuer: fmt.Sprintf("%s/runner", domain),
		secret:      secret,
	}
}

func (s *RunnersService) GetGameServer(ctx context.Context, tokenString string) (server.GameServer, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (any, error) {
		return []byte(s.secret), nil
	}, jwt.WithIssuer(s.tokenIssuer), jwt.WithValidMethods([]string{jwt.SigningMethodHS512.Alg()}))
	if err != nil {
		return nil, err
	}

	publicID, err := token.Claims.GetSubject()
	if err != nil {
		return nil, err
	}

	runner, err := s.repository.GetRunner(ctx, publicID)
	if err != nil {
		return nil, err
	}

	provider, ok := providers[runner.Type]
	if !ok {
		return nil, fmt.Errorf("unknown provider: %s", runner.Type)
	}

	return provider(ctx, runner.ProviderSpecificData)
}

func (s *RunnersService) Register(ctx context.Context, typ string, data map[string]string) (string, error) {
	provider, ok := providers[typ]
	if !ok {
		return "", fmt.Errorf("unknown provider: %s", typ)
	}

	gameServer, err := provider(ctx, data)
	if err != nil {
		return "", err
	}

	// Check validity
	if !gameServer.IsAvailable(ctx) {
		return "", errors.New("game server is not available")
	}

	publicID, err := s.repository.Create(ctx, typ, data)
	if err != nil {
		return "", err
	}

	claims := jwt.RegisteredClaims{
		Issuer:   s.tokenIssuer,
		Subject:  publicID,
		IssuedAt: jwt.NewNumericDate(time.Now()),
		ID:       uuid.NewString(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS512, claims)
	tokenString, err := token.SignedString([]byte(s.secret))
	if err != nil {
		return "", err
	}

	return tokenString, nil
}

// Use this token for development:
// eyJhbGciOiJIUzUxMiIsInR5cCI6IkpXVCJ9.eyJpc3MiOiJodHRwOi8vbG9jYWxob3N0OjgwMDAvcnVubmVyIiwic3ViIjoiZGVmYXVsdCIsImlhdCI6MTUxNjIzOTAyMiwiaWQiOiIxMWUxZDNmZC1kMjI2LTQ5YTctODYwNC0yMGY2YjY1YTU0NmMifQ.B6h54RQbxOAFi-wjqYcg9KAgaq6qF6a2QeyrtcFVLEyl6r7FLT4b-Or36lhnbQn_l0NyFDeM6mj8FkhHvS6FUg
