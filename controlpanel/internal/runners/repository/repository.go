package repository

import (
	"context"
	"errors"

	"github.com/kofuk/premises/controlpanel/internal/config"
	"github.com/uptrace/bun"
)

type RunnerRepository struct {
	db  *bun.DB
	cfg *config.Config
}

func NewRunnerRepository(db *bun.DB, cfg *config.Config) *RunnerRepository {
	// TODO: Persist credentials to DB and remove `cfg` from here
	return &RunnerRepository{
		db:  db,
		cfg: cfg,
	}
}

type Runner struct {
	ID                   string
	Type                 string
	ProviderSpecificData map[string]string
}

func (r *RunnerRepository) GetRunner(ctx context.Context, publicID string) (*Runner, error) {
	// TODO: Implement DB query to get the runner by publicID
	if publicID != "default" {
		return nil, errors.New("not found")
	}

	data := map[string]string{
		"user":             r.cfg.ConohaUser,
		"password":         r.cfg.ConohaPassword,
		"tenant_id":        r.cfg.ConohaTenantID,
		"name_tag":         r.cfg.ConohaNameTag,
		"identity_service": r.cfg.ConohaIdentityService,
		"compute_service":  r.cfg.ConohaComputeService,
		"image_service":    r.cfg.ConohaImageService,
		"volume_service":   r.cfg.ConohaVolumeService,
	}
	return &Runner{
		ID:                   publicID,
		Type:                 "Conoha",
		ProviderSpecificData: data,
	}, nil
}

func (r *RunnerRepository) Create(ctx context.Context, typ string, data map[string]string) (string, error) {
	return "default", nil
}
