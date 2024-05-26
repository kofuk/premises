package migrations

import (
	"context"

	"github.com/kofuk/premises/controlpanel/internal/db/model"
	"github.com/uptrace/bun"
)

func init() {
	Migrations.MustRegister(func(ctx context.Context, db *bun.DB) error {
		if _, err := db.NewCreateTable().IfNotExists().Model((*model.User)(nil)).ForeignKey(`("added_by_user_id") REFERENCES "users" ("id")`).Exec(ctx); err != nil {
			return err
		}

		// Make created_at and updated_at default to current_timestamp (for older schema)
		if _, err := db.ExecContext(ctx, "ALTER TABLE users ALTER COLUMN created_at SET DEFAULT CURRENT_TIMESTAMP"); err != nil {
			return err
		}
		if _, err := db.ExecContext(ctx, "ALTER TABLE users ALTER COLUMN updated_at SET DEFAULT CURRENT_TIMESTAMP"); err != nil {
			return err
		}

		// Make created_at and updated_at not null (for older schema)
		if _, err := db.ExecContext(ctx, "ALTER TABLE users ALTER COLUMN created_at SET NOT NULL"); err != nil {
			return err
		}
		if _, err := db.ExecContext(ctx, "ALTER TABLE users ALTER COLUMN updated_at SET NOT NULL"); err != nil {
			return err
		}

		return nil
	}, func(ctx context.Context, db *bun.DB) error {
		if _, err := db.NewDropTable().Model((*model.User)(nil)).Exec(ctx); err != nil {
			return err
		}
		return nil
	})
}
