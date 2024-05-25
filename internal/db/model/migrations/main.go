package migrations

import (
	"context"

	"github.com/uptrace/bun/migrate"
)

var Migrations = migrate.NewMigrations()

func Migrate(ctx context.Context, migrator *migrate.Migrator) error {
	if err := migrator.Init(ctx); err != nil {
		return err
	}

	migrator.Lock(ctx)
	defer migrator.Unlock(ctx)

	if _, err := migrator.Migrate(ctx); err != nil {
		return err
	}

	return nil
}
