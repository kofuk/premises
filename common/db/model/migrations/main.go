package migrations

import (
	"context"
	"log/slog"

	"github.com/uptrace/bun/migrate"
)

var Migrations = migrate.NewMigrations()

func Migrate(migrator *migrate.Migrator) error {
	slog.Info("Initializing bun migration")
	if err := migrator.Init(context.Background()); err != nil {
		return err
	}

	migrator.Lock(context.Background())
	defer migrator.Unlock(context.Background())

	group, err := migrator.Migrate(context.Background())
	if err != nil {
		return err
	}
	if group.IsZero() {
		slog.Info("No new migrations")
	} else {
		slog.Info("Migration completed", slog.String("to", group.String()))
	}

	return nil
}
