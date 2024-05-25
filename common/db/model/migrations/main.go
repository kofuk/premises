package migrations

import (
	"context"

	"github.com/uptrace/bun/migrate"
)

var Migrations = migrate.NewMigrations()

func Migrate(migrator *migrate.Migrator) error {
	if err := migrator.Init(context.Background()); err != nil {
		return err
	}

	migrator.Lock(context.Background())
	defer migrator.Unlock(context.Background())

	if _, err := migrator.Migrate(context.Background()); err != nil {
		return err
	}

	return nil
}
